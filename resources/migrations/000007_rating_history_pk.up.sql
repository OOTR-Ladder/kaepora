DROP TABLE "PlayerRatingHistory";

CREATE TABLE "PlayerRatingHistory" (
    "PlayerID"              blob(16) NOT NULL,
    "LeagueID"              blob(16) NOT NULL,
    "CreatedAt"             INT      NOT NULL,
    "RatingPeriodStartedAt" INT      NOT NULL,

    "Rating"     REAL NOT NULL,
    "Deviation"  REAL NOT NULL,
    "Volatility" REAL NOT NULL,

    PRIMARY KEY ("PlayerID", "LeagueID", "RatingPeriodStartedAt"),
    FOREIGN KEY(PlayerID) REFERENCES Player(ID) ON UPDATE CASCADE ON DELETE RESTRICT,
    FOREIGN KEY(LeagueID) REFERENCES League(ID) ON UPDATE CASCADE ON DELETE RESTRICT
);
