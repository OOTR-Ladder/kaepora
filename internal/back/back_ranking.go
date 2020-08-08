package back

import (
	"fmt"
	"kaepora/internal/util"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	glicko "github.com/zelenin/go-glicko2"
)

// updateRankingsForMatchSession updates every player in a League with the
// outcome from the matches in the session.
// The matches slice is an implementation detail and contains all matches of
// the session, since the caller already has the list we don't want to fetch
// them again.
func (b *Back) updateLeagueRankings(tx *sqlx.Tx, leagueID util.UUIDAsBlob, now time.Time) error {
	previousPeriodStart := util.TimeAsTimestamp(previousPeriodStart(now))
	currentPeriodStart := util.TimeAsTimestamp(currentPeriodStart(now))
	nextPeriodStart := util.TimeAsTimestamp(nextPeriodStart(now))
	log.Printf("debug: update league rankings for period %s to %s", currentPeriodStart.Time(), nextPeriodStart.Time())

	glickoPlayers, err := getGlickoPlayersForLeague(tx, leagueID, previousPeriodStart)
	if err != nil {
		return fmt.Errorf("unable to fetch ratings: %w", err)
	}
	log.Printf("debug: got %d ratings from previous period", len(glickoPlayers))

	matches, err := getMatchesByPeriod(tx, leagueID, currentPeriodStart, nextPeriodStart)
	if err != nil {
		return fmt.Errorf("unable to fetch matches for period: %w", err)
	}

	computePeriod(leagueID, matches, glickoPlayers)
	if err := b.updateRunningPeriodRatings(tx, leagueID, glickoPlayers); err != nil {
		return err
	}

	return b.closeRatingPeriod(tx, currentPeriodStart, leagueID, glickoPlayers)
}

func (b *Back) updateRunningPeriodRatings(
	tx *sqlx.Tx,
	leagueID util.UUIDAsBlob,
	glickoPlayers map[util.UUIDAsBlob]*glicko.Player,
) error {
	log.Printf("debug: updating %d PlayerRating entries", len(glickoPlayers))
	for playerID, glickoPlayer := range glickoPlayers {
		rating := NewPlayerRating(playerID, leagueID)
		glickoRating := glickoPlayer.Rating()

		rating.Rating = glickoRating.R()
		rating.Deviation = glickoRating.Rd()
		rating.Volatility = glickoRating.Sigma()

		if err := rating.upsert(tx); err != nil {
			return fmt.Errorf("unable to update rating: %w", err)
		}
	}

	return nil
}

func computePeriod(
	leagueID util.UUIDAsBlob,
	matches []Match,
	glickoPlayers map[util.UUIDAsBlob]*glicko.Player,
) {
	getGlickoPlayer := func(playerID, leagueID util.UUIDAsBlob) *glicko.Player {
		p, ok := glickoPlayers[playerID]
		if !ok {
			p = glicko.NewPlayer(NewPlayerRating(playerID, leagueID).GlickoRating())
			glickoPlayers[playerID] = p
		}
		return p
	}

	period := glicko.NewRatingPeriod()
	// Add players so Glicko-2 know to ensure inactive players decay.
	for k := range glickoPlayers {
		period.AddPlayer(glickoPlayers[k])
	}

	for k := range matches {
		p1 := getGlickoPlayer(matches[k].Entries[0].PlayerID, leagueID)
		p2 := getGlickoPlayer(matches[k].Entries[1].PlayerID, leagueID)

		switch matches[k].Entries[0].Outcome {
		case MatchEntryOutcomeWin:
			period.AddMatch(p1, p2, glicko.MATCH_RESULT_WIN)
		case MatchEntryOutcomeDraw:
			period.AddMatch(p1, p2, glicko.MATCH_RESULT_DRAW)
		case MatchEntryOutcomeLoss:
			period.AddMatch(p1, p2, glicko.MATCH_RESULT_LOSS)
		}
	}

	start := time.Now()
	period.Calculate()
	log.Printf(
		"info: recalculated leaderboard for %d matches and %d players in %s",
		len(matches), len(glickoPlayers),
		time.Since(start),
	)
}

// currentPeriodStart returns the previous monday at 00:00 UTC.
func currentPeriodStart(t time.Time) time.Time {
	t = t.UTC()

	if wd := t.Weekday(); wd == time.Sunday {
		t = t.AddDate(0, 0, -6)
	} else {
		t = t.AddDate(0, 0, -int(wd)+1)
	}

	return t.Truncate(24 * time.Hour)
}

// nextPeriodStart returns the next monday at 00:00 UTC.
func nextPeriodStart(t time.Time) time.Time {
	return currentPeriodStart(t).AddDate(0, 0, 7)
}

// previousPeriodStart returns the monday before the previous one.
func previousPeriodStart(t time.Time) time.Time {
	return currentPeriodStart(t).AddDate(0, 0, -7)
}

// deleteLeagueRankings removes the all ranking and ranking history of a given league.
func deleteLeagueRankings(tx *sqlx.Tx, leagueID util.UUIDAsBlob) error {
	if _, err := tx.Exec(
		`DELETE FROM "PlayerRatingHistory" WHERE LeagueID = ?`,
		leagueID,
	); err != nil {
		return err
	}

	if _, err := tx.Exec(
		`DELETE FROM "PlayerRating" WHERE LeagueID = ?`,
		leagueID,
	); err != nil {
		return err
	}

	return nil
}

// closeRatingPeriod writes the current rankings to PlayerRatingHistory, it can
// be called at any point in a period as rankings are upserted.
func (b *Back) closeRatingPeriod(
	tx *sqlx.Tx,
	currentPeriodStart util.TimeAsTimestamp,
	leagueID util.UUIDAsBlob,
	glickoPlayers map[util.UUIDAsBlob]*glicko.Player,
) error {
	log.Printf(
		"debug: closing period starting at %s, upsert history for %d players",
		currentPeriodStart.Time(),
		len(glickoPlayers),
	)

	for playerID, glickoPlayer := range glickoPlayers {
		rating := NewPlayerRating(playerID, leagueID)
		glickoRating := glickoPlayer.Rating()
		rating.SetRating(glickoRating)

		if err := rating.upsertHistory(tx, currentPeriodStart); err != nil {
			return fmt.Errorf("unable to insert rating history: %w", err)
		}
	}

	return nil
}

func (b *Back) Rerank(shortcode string) error {
	var (
		league          League
		firstMatchStart util.TimeAsTimestamp
	)
	start := time.Now()

	if err := b.transaction(func(tx *sqlx.Tx) (err error) {
		league, err = getLeagueByShortCode(tx, shortcode)
		if err != nil {
			return fmt.Errorf("unable to find league with shortcode '%s': %w", shortcode, err)
		}

		if err := deleteLeagueRankings(tx, league.ID); err != nil {
			return fmt.Errorf("unable to prune rankings: %w", err)
		}

		firstMatchStart, err = getFirstMatchStartOfLeague(tx, league.ID)
		if err != nil {
			return fmt.Errorf("unable to find first match of league: %w", err)
		}

		return nil
	}); err != nil {
		return err
	}

	firstPeriodStart := currentPeriodStart(firstMatchStart.Time())
	curPeriodEnd := nextPeriodStart(time.Now())
	log.Printf("debug: first match: %s (period %s)", firstMatchStart.Time(), firstPeriodStart)

	for i := firstPeriodStart; i.Before(curPeriodEnd); i = i.AddDate(0, 0, 7) {
		j := i // get out of range scope

		if err := b.transaction(func(tx *sqlx.Tx) (err error) {
			if err := b.updateLeagueRankings(tx, league.ID, j); err != nil {
				return fmt.Errorf("unable to update league rankings: %w", err)
			}

			return nil
		}); err != nil {
			return err
		}
	}

	log.Printf("info: recomputed rankings in %s", time.Since(start))

	return nil
}
