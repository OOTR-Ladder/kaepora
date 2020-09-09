-- The spoiler DB can be recreated from scratch anytime, hence why there will
-- be no migrations.

CREATE TABLE "ItemType" (
    "ID"   integer NOT NULL PRIMARY KEY AUTOINCREMENT,
    "Name" text    NOT NULL
);

CREATE TABLE "Item" (
    "ID"     integer NOT NULL PRIMARY KEY AUTOINCREMENT,
    "TypeID" integer NOT NULL,
    "Name"   text    NOT NULL,

    FOREIGN KEY ("TypeID") REFERENCES "ItemType" ("ID") ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE "Region" (
  "ID"   integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  "Name" text    NOT NULL
);

CREATE TABLE "Location" (
    "ID"       integer NOT NULL  PRIMARY KEY AUTOINCREMENT,
    "RegionID" integer NOT NULL,
    "Name"     text    NOT NULL,
    "HintType" integer NOT NULL, -- 0: not hinted, 1: sometimes, 2: always

    FOREIGN KEY ("RegionID") REFERENCES "Region" ("ID") ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE "Seed" (
  "ID"   integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  "Seed" text    NOT NULL
);

CREATE TABLE "ItemEntry" (
    "ID"          integer NOT NULL  PRIMARY KEY AUTOINCREMENT,
    "SeedID"      integer NOT NULL,
    "LocationID"  integer NOT NULL,
    "ItemID"      integer NOT NULL,
    "Sphere"      integer NOT NULL,
    "IsWOTH"      integer NOT NULL,
    "Price"       integer NULL, -- optional, for scrubs and shops
    "ModelItemID" integer NULL, -- optional, for ice traps

    FOREIGN KEY ("SeedID")      REFERENCES "Seed"     ("ID") ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY ("LocationID")  REFERENCES "Location" ("ID") ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY ("ItemID")      REFERENCES "Item"     ("ID") ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY ("ModelItemID") REFERENCES "Item"     ("ID") ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE "GossipStone" (
  "ID"   integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  "Name" text    NOT NULL
);

CREATE TABLE "HintEntry" (
    "ID"            integer NOT NULL PRIMARY KEY AUTOINCREMENT,
    "SeedID"        integer NOT NULL,
    "GossipStoneID" integer NOT NULL,
    "LocationID"    integer NULL, -- Location & Item => Always/Sometimes hint
    "ItemID"        integer NULL,
    "RegionID"      integer NULL, -- Region & IsWOTH => WoTH/Barren hint
    "IsWOTH"        integer NULL, -- 0: Barren, 1: WoTH

    FOREIGN KEY ("GossipStoneID") REFERENCES "GossipStone" ("ID") ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY ("RegionID")      REFERENCES "Region"      ("ID") ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY ("ItemID")        REFERENCES "Item"        ("ID") ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY ("LocationID")    REFERENCES "Location"    ("ID") ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY ("SeedID")        REFERENCES "Seed"        ("ID") ON DELETE CASCADE ON UPDATE CASCADE
);
