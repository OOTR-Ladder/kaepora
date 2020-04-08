CREATE TABLE "Game" (
    "ID"        blob(16) NOT   NULL,
    "CreatedAt" INT      NOT   NULL,
    "Name"      TEXT     NOT   NULL,
    "Generator" TEXT     NULL, -- matches an hardcoded name

    PRIMARY KEY ("ID")
);

CREATE TABLE "League" (
    "ID"        blob(16) NOT NULL,
    "CreatedAt" INT      NOT NULL,
    "Name"      TEXT     NOT NULL,
    "ShortCode" TEXT     NOT NULL,
    "GameID"    blob(16) NOT NULL,
    "Settings"  TEXT     NOT NULL, -- tied to the parent Game generator

    -- JSON, eg. {"Mon": ["20:00 Europe/Paris"]}, the Schedule is only used for
    -- generating MatchSession, all runtime race stuff is done using the
    -- resulting MatchSession.
    "Schedule" TEXT      NOT NULL,

    PRIMARY KEY ("ID"),
    FOREIGN KEY(GameID) REFERENCES Game(ID) ON UPDATE CASCADE ON DELETE RESTRICT
);

CREATE UNIQUE INDEX idx_unique_ShortCode ON League (ShortCode);

CREATE TABLE "Player" (
    "ID"        blob(16) NOT NULL,
    "CreatedAt" INT      NOT NULL,
    "Name"      TEXT     NOT NULL,
    "DiscordID" TEXT     NULL,

    PRIMARY KEY ("ID")
);

CREATE UNIQUE INDEX idx_unique_Name      ON Player (Name);
CREATE UNIQUE INDEX idx_unique_DiscordID ON Player (DiscordID);

CREATE TABLE "PlayerRating" (
    "PlayerID"  blob(16) NOT NULL,
    "LeagueID"  blob(16) NOT NULL,
    "CreatedAt" INT      NOT NULL,

    -- Glicko-2 https://www.glicko.net/glicko/glicko2.pdf
    "Rating"     REAL NOT NULL,
    "Deviation"  REAL NOT NULL,
    "Volatility" REAL NOT NULL,

    PRIMARY KEY ("PlayerID", "LeagueID"),
    FOREIGN KEY(PlayerID) REFERENCES Player(ID) ON UPDATE CASCADE ON DELETE RESTRICT,
    FOREIGN KEY(LeagueID) REFERENCES League(ID) ON UPDATE CASCADE ON DELETE RESTRICT
);

CREATE TABLE "MatchSession" (
    "ID"        blob(16) NOT NULL,
    "LeagueID"  blob(16) NOT NULL,

    "CreatedAt" INT  NOT NULL,
    "StartDate" TEXT NOT NULL,

    -- 0: MatchSessionWaiting,    1: MatchSessionJoinable,
    -- 2: MatchSessionPreparing,  3: MatchSessionInProgress,
    -- 4: MatchSessionClosed
    "Status" INT NOT NULL,

    -- JSON array of Player.ID that registered for the session
    -- Becomes readonly when reaching MatchSessionPreparing
    "PlayerIDs" TEXT NOT NULL,

    PRIMARY KEY ("ID"),
    FOREIGN KEY(LeagueID) REFERENCES League(ID) ON UPDATE CASCADE ON DELETE RESTRICT
);

CREATE INDEX idx_Status ON MatchSession (Status);

CREATE TABLE "Match" (
    "ID"             blob(16) NOT NULL,
    "LeagueID"       blob(16) NOT NULL,
    "MatchSessionID" blob(16) NOT NULL,

    "CreatedAt" INT NOT NULL,
    "StartedAt" INT NULL,
    "EndedAt"   INT NULL,

    "Generator" TEXT NOT NULL, -- parent League->Game.Generator at creation time
    "Settings"  TEXT NOT NULL, -- parent League.Settings at creation time
    "Seed"      TEXT NOT NULL, -- CSPRNG, format depends on League->Game.Generator

    PRIMARY KEY ("ID"),
    FOREIGN KEY(LeagueID)       REFERENCES League(ID)       ON UPDATE CASCADE ON DELETE RESTRICT
    FOREIGN KEY(MatchSessionID) REFERENCES MatchSession(ID) ON UPDATE CASCADE ON DELETE RESTRICT
);

CREATE TABLE "MatchEntry" (
    "MatchID"   blob(16) NOT NULL,
    "PlayerID"  blob(16) NOT NULL,
    "CreatedAt" INT      NOT NULL,
    "StartedAt" INT      NULL,
    "EndedAt"   INT      NULL,

    -- 0: MatchEntryStatusWaiting,  1: MatchEntryStatusInProgress,
    -- 2: MatchEntryStatusFinished, 3: MatchEntryStatusForfeit
    "Status"    INT      NOT NULL DEFAULT 0,

    -- -1: MatchOutcomeLoss, 0: MatchOutcomeDraw, 1: MatchOutcomeWin
    "Outcome"   INT      NOT NULL, -- only valid if Status > 1

    PRIMARY KEY ("MatchID", "PlayerID"),
    FOREIGN KEY(MatchID)  REFERENCES Match(ID)  ON UPDATE CASCADE ON DELETE RESTRICT,
    FOREIGN KEY(PlayerID) REFERENCES Player(ID) ON UPDATE CASCADE ON DELETE RESTRICT
);
