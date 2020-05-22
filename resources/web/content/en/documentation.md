## What are ladder races?
Ladder races are regular one versus one races where you don't know who your
opponent is and you have to finish a game of _Ocarina of Time: Randomizer_
(OoTR) in the shortest possible time.  
When all races of a session are completed the leaderboard is updated and the
top three players gain bragging rights.

## Signing up and racing
To participate you will need to [read the rules](/rules) and register yourself
by sending the `!register` command in the `#ladder-signup` channel of the [OoTR
Discord server](https://discord.gg/yZtdURz). The bot _Kaepora_ will answer you
via private message and this is also where you should type any other command.

<div class="message is-warning">
    <div class="message-body">
        <p>You need to <em>allow direct messages from server members</em> in
        your Discord <em>Privacy & Safety</em> settings, otherwise the bot will
        not be able to message you.</p>
    </div>
</div>

When a race is coming up, _Kaepora_ will announce it in dedicated channels of
the _IR Ladder_ category. After the announce is made you can send _Kaepora_ the
`!join` command followed by the corresponding league short name. eg. for the
_Standard_ league you would send `!join std`.  
If you no longer wish to race you can send the `!cancel` command. This will not
affect your ranking.

<div class="message is-warning">
    <div class="message-body">
        <p>If there is an odd number of players, the last person to join will
        be kicked out of the race.</p>
    </div>
</div>

**Fifteen minutes before the race** Kaepora will send you a link to your _seed_
on [ootrandomizer.com](https://ootrandomizer.com). There you can patch your own
ROM and install it on your Wii or emulator
([Retroarch](https://www.retroarch.com/) with the ParaLLEl core is strongly
recommended, Bizhawk and Project64 1.6-1.7 are also allowed).

The race starts after _Kaepora_ gives everyone a countdown on the channel
dedicated to the league you registered for (eg. `#ladder-league-standard`).  
All races start at the same time, if you are late the race will not be delayed
and your opponent will be at an advantage.

When you finish the game (either when dealing the death blow on Ganon or by
finding the last Triforce fragment in some _Shuffled Settings_ races) you need
to send `!done` to _Kaepora_. The bot will then answer with your time, the name
of your opponent, their time if they finished first, and your _spoiler log_
(which you can also access by using the original link to your seed).

If for any reason you cannot finish your game, send `!forfeit`. A forfeit is an
automatic loss unless your opponent also forfeited, which incurs a tie.

<div class="message is-info">
    <div class="message-header"><p>A quick recap:</p></div>
    <div class="message-body">
        <ul>
        <li><code>!help</code> should be the only command you need to remember and probably the first one to try out.</li>
        <li><code>!register</code> first.</li>
        <li><code>!join &lt;short name&gt;</code> when a race is open for registration.</li>
        <li><code>!done</code> on the final blow or <code>!forfeit</code>.
    </ul>
</div>
</div>

## Leaderboards
Each league has its own independent leaderboard which is generated using the
[Glicko-2 rating system][1] using a seven day period.  

This system has a few particularities:

 - There is no single number that represents your skill level, instead you have
   a range where your skill is estimated to be in with 95 % confidence.
 - The reward (or penalty) for winning (or losing) a race is proportional to
   the skill gap between the two players.
 - You need to complete a few races to lower your _rating deviation_ and reach
   the required threshold to appear in the leaderboards.
 - Not racing for a week will slightly decrease your rating and increase your
   _rating deviation_.

[1]: https://en.wikipedia.org/wiki/Glicko_rating_system

## Leagues
### Standard (`std`)
The _Standard_ league is the default league where all players, good and bad,
should try their luck and skills.  
The ruleset used on this league is the [OoTR Standard Racing Ruleset (SRR)][2],
seeds are generated using the latest _minor_ version of the randomizer.

There are multiple sessions per day and you are free to join any that fit your
schedule.

Each versus has its own seed.

[2]: https://wiki.ootrandomizer.com/index.php?title=Standard

### Shuffled settings (`shu`)
The _Shuffled Settings_ league uses a different kind of seed generation where
the settings of the randomizer itself are randomized.

Settings are not _fully_ randomized: the algorithm iterates through a list of
_weighted_ setting values to pick one at random and adds its _cost_ to the
running total.  
Once the running total reaches the budgeted cost, the algorithms stops and
applies the new settings on top of the standard settings.

Costs and weights of setting values were designed to allow for interesting
seeds without having to delve into the pure madness of a _fullsanity_.

Each versus has its own seed and set of settings.

## Contributing
The ladder is a collaborative effort released under the MIT license.  
You can contribute by sending well-written pull requests to the [Kaepora][3]
repository which contains the sources for both the bot and the website.

[3]: https://github.com/OOTR-Ladder/kaepora
