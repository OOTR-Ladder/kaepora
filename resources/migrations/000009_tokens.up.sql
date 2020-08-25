CREATE TABLE "Token" (
  "ID" blob(16) NOT NULL,
  "CreatedAt" integer NOT NULL,
  "ExpiresAt" integer NOT NULL,
  "PlayerID" blob NOT NULL,
  FOREIGN KEY ("PlayerID") REFERENCES "Player" ("ID") ON DELETE CASCADE
);
