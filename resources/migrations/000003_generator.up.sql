ALTER TABLE "League" ADD "Generator" text NOT NULL DEFAULT '';
UPDATE League SET Generator = (
    SELECT Game.Generator FROM Game WHERE League.GameID = Game.ID
);

CREATE TABLE "backup_Game" (
  "ID" blob NOT NULL,
  "CreatedAt" integer NOT NULL,
  "Name" text NOT NULL,
  PRIMARY KEY ("ID")
);
INSERT INTO "backup_Game" ("ID", "CreatedAt", "Name") SELECT "ID", "CreatedAt", "Name" FROM "Game";
DROP TABLE "Game";

CREATE TABLE "Game" (
  "ID" blob NOT NULL,
  "CreatedAt" integer NOT NULL,
  "Name" text NOT NULL,
  PRIMARY KEY ("ID")
);
INSERT INTO "Game" ("ID", "CreatedAt", "Name") SELECT "ID", "CreatedAt", "Name" FROM "backup_Game";
DROP TABLE "backup_Game";
