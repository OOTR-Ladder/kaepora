PRAGMA foreign_keys = OFF;

CREATE TABLE "backup_Player" (
  "ID" blob NOT NULL,
  "CreatedAt" integer NOT NULL,
  "Name" text NOT NULL,
  "DiscordID" text NULL,
  PRIMARY KEY ("ID")
);
INSERT INTO "backup_Player" ("ID", "CreatedAt", "Name", "DiscordID") SELECT "ID", "CreatedAt", "Name", "DiscordID" FROM "Player";

DROP TABLE "Player";
ALTER TABLE "backup_Player" RENAME TO "Player";
CREATE UNIQUE INDEX "idx_unique_DiscordID" ON "Player" ("DiscordID");
CREATE UNIQUE INDEX "idx_unique_Name" ON "Player" ("Name");

PRAGMA foreign_keys = ON;
