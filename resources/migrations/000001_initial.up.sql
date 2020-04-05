CREATE TABLE "Game" (
    "ID"        blob(16) NOT   NULL,
    "CreatedAt" INT      NOT   NULL  DEFAULT CURRENT_TIMESTAMP,
    "Name"      TEXT     NOT   NULL,
    "Generator" TEXT     NULL, -- matches an hardcoded name

    PRIMARY KEY ("ID")
);

CREATE TABLE "League" (
    "ID"        blob(16) NOT NULL,
    "CreatedAt" INT      NOT NULL  DEFAULT CURRENT_TIMESTAMP,
    "Name"      TEXT     NOT NULL,
    "GameID"    blob(16) NOT NULL,
    "Settings"  TEXT     NOT NULL, -- tied to the parent Game generator

    PRIMARY KEY ("ID"),
    FOREIGN KEY(GameID) REFERENCES Game(ID) ON UPDATE CASCADE ON DELETE RESTRICT
);

CREATE TABLE "Player" (
    "ID"        blob(16) NOT NULL,
    "CreatedAt" INT      NOT NULL  DEFAULT CURRENT_TIMESTAMP,
    "Name"      TEXT     NOT NULL,

    PRIMARY KEY ("ID")
);

CREATE TABLE "PlayerRating" (
    "PlayerID"  blob(16) NOT NULL,
    "LeagueID"  blob(16) NOT NULL,
    "CreatedAt" INT      NOT NULL  DEFAULT CURRENT_TIMESTAMP,

    -- Glicko-2 https://www.glicko.net/glicko/glicko2.pdf
    "Rating"     REAL NOT NULL,
    "Deviation"  REAL NOT NULL,
    "Volatility" REAL NOT NULL,

    PRIMARY KEY ("PlayerID", "LeagueID"),
    FOREIGN KEY(PlayerID) REFERENCES Player(ID) ON UPDATE CASCADE ON DELETE RESTRICT,
    FOREIGN KEY(LeagueID) REFERENCES League(ID) ON UPDATE CASCADE ON DELETE RESTRICT
);

CREATE TABLE "Match" (
    "ID"        blob(16) NOT NULL,
    "LeagueID"  blob(16) NOT NULL,
    "CreatedAt" INT      NOT NULL  DEFAULT CURRENT_TIMESTAMP,
    "StartedAt" INT          NULL,
    "EndedAt"   INT          NULL,
    "Generator" TEXT     NOT NULL, -- parent League->Game.Generator at creation time
    "Settings"  TEXT     NOT NULL, -- parent League.Settings at creation time
    "Seed"      TEXT     NOT NULL, -- CSPRNG, format depends on League->Game.Generator

    PRIMARY KEY ("ID"),
    FOREIGN KEY(LeagueID) REFERENCES League(ID) ON UPDATE CASCADE ON DELETE RESTRICT
);

CREATE TABLE "MatchEntry" (
    "MatchID"   blob(16) NOT NULL,
    "PlayerID"  blob(16) NOT NULL,
    "CreatedAt" INT      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "StartedAt" INT      NULL,
    "EndedAt"   INT      NULL,
    "Status"    INT      NOT NULL DEFAULT 0, -- 0: waiting, 1: in progress, 2: forfeit (loss), 3: canceled (no penalty)
    "Outcome"   INT      NULL,               -- -1: loss, 0: draw, 1: win

    PRIMARY KEY ("MatchID", "PlayerID"),
    FOREIGN KEY(MatchID)  REFERENCES Match(ID)  ON UPDATE CASCADE ON DELETE RESTRICT,
    FOREIGN KEY(PlayerID) REFERENCES Player(ID) ON UPDATE CASCADE ON DELETE RESTRICT
);
