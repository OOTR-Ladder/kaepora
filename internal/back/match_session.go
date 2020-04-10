package back

import (
	"kaepora/internal/util"
	"sort"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

const (
	// joinable after T+offset (mind the negative offsets)
	MatchSessionJoinableAfterOffset = -1 * time.Hour
	// player receive seeds at T+offset and can no longer join/cancel
	MatchSessionPreparationOffset = -15 * time.Minute
)

type MatchSessionStatus int

const (
	MatchSessionStatusWaiting    MatchSessionStatus = 0 // waiting for StartDate - 30m
	MatchSessionStatusJoinable   MatchSessionStatus = 1 // waiting for runners to join
	MatchSessionStatusPreparing  MatchSessionStatus = 2 // runners setting up race
	MatchSessionStatusInProgress MatchSessionStatus = 3 // runners still racing
	MatchSessionStatusClosed     MatchSessionStatus = 4 // everyone finished
)

type MatchSession struct {
	ID        util.UUIDAsBlob
	LeagueID  util.UUIDAsBlob
	CreatedAt util.TimeAsTimestamp
	StartDate util.TimeAsDateTimeTZ
	Status    MatchSessionStatus
	PlayerIDs util.UUIDArrayAsJSON
}

func (s *MatchSession) GetPlayerIDs() []uuid.UUID {
	return s.PlayerIDs.Slice()
}

func (s *MatchSession) HasPlayerID(needle uuid.UUID) bool {
	for _, v := range s.GetPlayerIDs() {
		if v == needle {
			return true
		}
	}

	return false
}

func NewMatchSession(leagueID util.UUIDAsBlob, startDate time.Time) MatchSession {
	return MatchSession{
		ID:        util.NewUUIDAsBlob(),
		LeagueID:  leagueID,
		CreatedAt: util.TimeAsTimestamp(time.Now()),
		StartDate: util.TimeAsDateTimeTZ(startDate),
		Status:    MatchSessionStatusWaiting,
		PlayerIDs: nil,
	}
}

func getMatchSessionByStartDate(tx *sqlx.Tx, leagueID util.UUIDAsBlob, startDate time.Time) (MatchSession, error) {
	var ret MatchSession
	query := `SELECT * FROM MatchSession WHERE MatchSession.LeagueID = ? AND MatchSession.StartDate = ? LIMIT 1`
	if err := tx.Get(&ret, query, leagueID, util.TimeAsDateTimeTZ(startDate)); err != nil {
		return MatchSession{}, err
	}

	return ret, nil
}

func getMatchSessionByID(tx *sqlx.Tx, id util.UUIDAsBlob) (MatchSession, error) {
	var ret MatchSession
	query := `SELECT * FROM MatchSession WHERE MatchSession.ID = ? LIMIT 1`
	if err := tx.Get(&ret, query, id); err != nil {
		return MatchSession{}, err
	}

	return ret, nil
}

// nolint:interfacer
func getPlayerActiveSession(tx *sqlx.Tx, playerID uuid.UUID) (MatchSession, error) {
	var ret MatchSession
	query := `
        SELECT * FROM MatchSession
        WHERE MatchSession.Status IN(?, ?, ?) AND
            PlayerIDs LIKE ?
        ORDER BY MatchSession.StartDate ASC
        LIMIT 1`

	if err := tx.Get(
		&ret, query,
		MatchSessionStatusJoinable,
		MatchSessionStatusPreparing,
		MatchSessionStatusInProgress,
		`%"`+playerID.String()+`"%`,
	); err != nil {
		return MatchSession{}, err
	}

	return ret, nil
}

func getNextMatchSessionForLeague(tx *sqlx.Tx, leagueID util.UUIDAsBlob) (MatchSession, error) {
	var ret MatchSession
	query := `
        SELECT * FROM MatchSession
        WHERE MatchSession.LeagueID = ? AND
              DATETIME(MatchSession.StartDate) > DATETIME(?)
        ORDER BY MatchSession.StartDate ASC
        LIMIT 1`

	if err := tx.Get(
		&ret, query,
		leagueID,
		util.TimeAsDateTimeTZ(time.Now()),
	); err != nil {
		return MatchSession{}, err
	}

	return ret, nil
}

func getNextJoinableMatchSessionForLeague(tx *sqlx.Tx, leagueID util.UUIDAsBlob) (MatchSession, error) {
	var ret MatchSession
	query := `
        SELECT * FROM MatchSession
        WHERE MatchSession.LeagueID = ? AND
              DATETIME(MatchSession.StartDate) > DATETIME(?) AND
              MatchSession.Status = ?
        ORDER BY MatchSession.StartDate ASC
        LIMIT 1`

	if err := tx.Get(
		&ret, query,
		leagueID,
		util.TimeAsDateTimeTZ(time.Now()),
		MatchSessionStatusJoinable,
	); err != nil {
		return MatchSession{}, err
	}

	return ret, nil
}

func (s *MatchSession) Insert(tx *sqlx.Tx) error {
	query, args, err := squirrel.Insert("MatchSession").SetMap(squirrel.Eq{
		"ID":        s.ID,
		"CreatedAt": s.CreatedAt,
		"LeagueID":  s.LeagueID,
		"StartDate": s.StartDate,
		"Status":    s.Status,
		"PlayerIDs": s.PlayerIDs,
	}).ToSql()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}

	return nil
}

// AddPlayerID registers a player for the session, entries are sorted and deduplicated.
func (s *MatchSession) AddPlayerID(uuids ...uuid.UUID) {
	s.PlayerIDs = append(s.PlayerIDs, uuids...)
	sort.Sort(util.SortByUUID(s.PlayerIDs))

	j := 0
	for i := 1; i < len(s.PlayerIDs); i++ {
		if s.PlayerIDs[j] == s.PlayerIDs[i] {
			continue
		}
		j++

		s.PlayerIDs[j] = s.PlayerIDs[i]
	}
	s.PlayerIDs = s.PlayerIDs[:j+1]
}

func (s *MatchSession) RemovePlayerID(uuid uuid.UUID) {
	for k := range s.PlayerIDs {
		if s.PlayerIDs[k] == uuid {
			s.PlayerIDs[k] = s.PlayerIDs[len(s.PlayerIDs)-1]
			s.PlayerIDs = s.PlayerIDs[:len(s.PlayerIDs)-1]
			return
		}
	}
}

func (s *MatchSession) CanCancel() error {
	if s.Status == MatchSessionStatusWaiting {
		// unreachable
		return util.ErrPublic("you can't cancel a race that is not yet open to join")
	}

	if s.Status != MatchSessionStatusJoinable {
		return util.ErrPublic("you can't cancel a race that is not in its joinable phase")
	}

	deadline := s.StartDate.Time().Add(MatchSessionPreparationOffset)
	if time.Now().After(deadline) {
		return util.ErrPublic("this race is no longer cancellable, you will have to `!forfeit`")
	}

	return nil
}

func (s *MatchSession) CanForfeit() error {
	if err := s.CanCancel(); err == nil {
		return util.ErrPublic("you can `!cancel` the current race wihout taking a loss!")
	}

	if s.Status == MatchSessionStatusClosed {
		return util.ErrPublic("you can't cancel a race that has already finished")
	}

	if s.Status != MatchSessionStatusPreparing && s.Status != MatchSessionStatusInProgress {
		return util.ErrPublic("you can't cancel a race that has not started or is not in its preparation phase")
	}

	return nil
}

func (s *MatchSession) update(tx *sqlx.Tx) error {
	query, args, err := squirrel.Update("MatchSession").SetMap(squirrel.Eq{
		"StartDate": s.StartDate,
		"Status":    s.Status,
		"PlayerIDs": s.PlayerIDs,
	}).
		Where("MatchSession.ID = ?", s.ID).
		ToSql()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}

	return nil
}
