ALTER TABLE "Game" ADD "Generator" text NULL;
UPDATE Game SET Generator = (
    SELECT League.Generator FROM League WHERE League.GameID = Game.ID
);

PRAGMA foreign_keys = OFF;

CREATE TABLE "backup_League" (
  "ID" blob NOT NULL,
  "CreatedAt" integer NOT NULL,
  "Name" text NOT NULL,
  "ShortCode" text NOT NULL,
  "GameID" blob NOT NULL,
  "Settings" text NOT NULL,
  "Schedule" text NOT NULL,
  "AnnounceDiscordChannelID" text NULL
);
INSERT INTO "backup_League" ("ID", "CreatedAt", "Name", "ShortCode", "GameID", "Settings", "Schedule", "AnnounceDiscordChannelID") SELECT "ID", "CreatedAt", "Name", "ShortCode", "GameID", "Settings", "Schedule", "AnnounceDiscordChannelID" FROM "League";
DROP TABLE "League";

CREATE TABLE "League" (
  "ID" blob NOT NULL,
  "CreatedAt" integer NOT NULL,
  "Name" text NOT NULL,
  "ShortCode" text NOT NULL,
  "GameID" blob NOT NULL,
  "Settings" text NOT NULL,
  "Schedule" text NOT NULL,
  "AnnounceDiscordChannelID" text NULL,
  PRIMARY KEY ("ID"),
  FOREIGN KEY ("GameID") REFERENCES "Game" ("ID") ON DELETE RESTRICT ON UPDATE CASCADE
);

CREATE UNIQUE INDEX "idx_unique_ShortCode" ON "League" ("ShortCode");

INSERT INTO "League" ("ID", "CreatedAt", "Name", "ShortCode", "GameID", "Settings", "Schedule", "AnnounceDiscordChannelID") SELECT "ID", "CreatedAt", "Name", "ShortCode", "GameID", "Settings", "Schedule", "AnnounceDiscordChannelID" FROM "backup_League";
DROP TABLE "backup_League";

PRAGMA foreign_keys = ON;
