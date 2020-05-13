## Introduction
With the growing popularity of the *Ocarina of Time Randomizer* (OoTR)
community, it's harder for players to know their true level. Moreover, we have
heard some wishes to see a new racing form in the community.

Some other communities have answered to similar demands, like the *A Link To
The Past Randomizer* community.

Seeing those initiatives, we have decided to set up a new way to enjoy all the
randomizer has to offer.  
We are happy to introduce to you our OoTR Ladder race system.

### How does it work?
The ranking system used for OoTR ladder is
[_Glicko-2_](https://en.wikipedia.org/wiki/Glicko_rating_system).
Therefore, each player's score evolution is determined by one-on-one matches.
Points will be added or removed depending on its performance in such a way that
the points of each player, assumed to be of equal level, are getting closer to
equality.

The amount of points won or lost by the player will be mostly determined by the
ranking gap between them and their opponent, along with everyone's past
performance.

No player knows their opponent. The identity of each member of the pair and
their respective times won't be revealed until each of them has finished their
seed.

At the end of each session, the leaderboard will be updated according to every
one-on-one result.

### Who can join OoTR Ladder races?
Every player wanting to progress in *Ocarina of Time Randomizer* is free to
join the scheduled races.

We want to keep this important concept of accessibility. Regardless of your
performance, you will have an opponent of your level to allow you to give the
best of yourself while avoiding this feeling of noticeable imbalance that can
emerge from some classic races.

## Technical aspect
### OoTR ladder leagues
Each OoTR league will follow the **Standard** ruleset. If you want more
informations about all the allowed tricks by the ruleset, we invite you to
visit [this page](https://wiki.ootrandomizer.com/index.php?title=Standard).

Each league will have its own theme. For now, only the Standard league is
available.  
It will be the most representative league for any player wishing to know their
true level. The league will only showcase seeds in **Weekly Settings**,
settings that will continue to evolve along with the community.

### Running a OoTR Ladder race
#### Prerequisites
**All race registrations are done in the OoTR Ladder Discord server.** You can
find the link in the upper right corner if this page.

When arriving in the server, you will have to input some commands in order to
be able to join a race. The `!help` command of the bot developed for this
purpose will provide to you all the necessary tools.

Here is an exhaustive list of all the bot's commands allowing you to fulfill
all the prerequisites to join a race:

- `!help` : Display a message grouping all available bot commands
- `!leagues` : Display the list of leagues offered on OoTR Ladder
- `!leaderboard <League>` : Show leaderboards for the given league
- `!recap <League>` : Show the 1v1 results for the current session
- `!register` : Allow member to join races
- `!register <Pseudo>` : Allow member to join races with a custom nickname
- `!rename <Pseudo>` : Change your nickname on OoTR Ladder

#### Join a room
After you have registered to take part in your first race with the `!register` command, you'll be able to look for future announcements from the Discord bot for the league you're interested in.
Each league has its own announcement channel.

Each race takes place in the form of sessions. Every session opens up 1 hour
before the race start. To join it, you just have to send the command specified
by the announcement.

If you have joined a session but you can't guarantee your participation anymore
because of whatever reason, you have the possibility to leave the session
***until 15 minutes*** before its start with the `!cancel` command. No penalty
will be assigned to you.

When we are in the last **15 minutes**, the session will be locked : you can't cancel your participation anymore and players can no longer join during this period.
All registered players will be assigned an opponent and the seed corresponding to your one-on-one match will be sent to you by Discord PM.
You'll then have 15 minutes to prepare your setup.

As you can see, **each one-on-one match in a session will have its own seed**:
only your opponent will be given the same OoTR seed as yours.

Some reminders will be sent in the corresponding league's channel before the
beginning of your match:

1. 1 minute before the beginning
2. 30 seconds before the beginning
3. 10 seconds before the beginning

A countdown will start in the last 5 seconds before the bot gives the “Go !” to
all registered players.

<div class="notification is-danger is-light">
<strong>In the specific situation of an odd number of registered
players</strong> in the session … it is the hard law of "first come, first
served" that applies. As it's simply impossible to play against yourself, the
last player's registration will be automatically canceled. Obviously, no
penalty will be attributed to this player.
</div>

Here is an exhaustive list of all the useful bot's commands for preparing your
OoTR Ladder races :

- `!join <League>` : Join a session for the desired league
- `!cancel` : Cancel your registration (Unusable within the last 15 minutes before the match).

#### Race start
The bot announces the start. From this moment, we can only wish you good luck
and **scrupulously follow** all the current OoTR Ladder rules that you can find
in the [dedicated rules page](/rules).

We encourage each participant to finish their seeds and to forfeit as little as
possible. Not giving you your opponent's identity is a choice we think
compatible with this philosophy.

When the pig is finally dead, you'll just have to send the `!done` command
either by PM to the bot, either in the **#talk-to-kaepora** channel. In the
case where you unfortunately can't finish, use the `!forfeit` command.

When the last member of the pair has sent one of the commands, your one-on-one
match is officially over and the identity of your opponent as well as your
respective times are revealed.

**The leaderboard and your ladder points will only be obtained once all
one-on-one matches have been completed.**

## Useful tools for your races
### Trackers
- [EmoTracker](https://emotracker.net/)
- [LinSo Tracker](https://pastebin.com/vYrNGweu)
- [Track-OOT](https://track-oot.net/)
