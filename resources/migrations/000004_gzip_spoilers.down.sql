PRAGMA foreign_keys = OFF;

CREATE TABLE "backup_Match" (
  "ID" blob NOT NULL,
  "LeagueID" blob NOT NULL,
  "MatchSessionID" blob NOT NULL,
  "CreatedAt" integer NOT NULL,
  "StartedAt" integer NULL,
  "EndedAt" integer NULL,
  "Generator" text NOT NULL,
  "Settings" text NOT NULL,
  "Seed" text NOT NULL,
  "SpoilerLog" text NOT NULL DEFAULT '',
  PRIMARY KEY ("ID"),
  FOREIGN KEY ("MatchSessionID") REFERENCES "MatchSession" ("ID") ON DELETE RESTRICT ON UPDATE CASCADE,
  FOREIGN KEY ("LeagueID") REFERENCES "League" ("ID") ON DELETE RESTRICT ON UPDATE CASCADE
);

INSERT INTO "backup_Match" ("ID", "LeagueID", "MatchSessionID", "CreatedAt", "StartedAt", "EndedAt", "Generator", "Settings", "Seed", "SpoilerLog") SELECT "ID", "LeagueID", "MatchSessionID", "CreatedAt", "StartedAt", "EndedAt", "Generator", "Settings", "Seed", "SpoilerLog" FROM "Match";
DROP TABLE "Match";

CREATE TABLE "Match" (
  "ID" blob NOT NULL,
  "LeagueID" blob NOT NULL,
  "MatchSessionID" blob NOT NULL,
  "CreatedAt" integer NOT NULL,
  "StartedAt" integer NULL,
  "EndedAt" integer NULL,
  "Generator" text NOT NULL,
  "Settings" text NOT NULL,
  "Seed" text NOT NULL,
  "SpoilerLog" text NOT NULL DEFAULT '',
  PRIMARY KEY ("ID"),
  FOREIGN KEY ("MatchSessionID") REFERENCES "MatchSession" ("ID") ON DELETE RESTRICT ON UPDATE CASCADE,
  FOREIGN KEY ("LeagueID") REFERENCES "League" ("ID") ON DELETE RESTRICT ON UPDATE CASCADE
);

INSERT INTO "Match" ("ID", "LeagueID", "MatchSessionID", "CreatedAt", "StartedAt", "EndedAt", "Generator", "Settings", "Seed", "SpoilerLog") SELECT "ID", "LeagueID", "MatchSessionID", "CreatedAt", "StartedAt", "EndedAt", "Generator", "Settings", "Seed", "SpoilerLog" FROM "backup_Match";
DROP TABLE "backup_Match";

PRAGMA foreign_keys = ON;
