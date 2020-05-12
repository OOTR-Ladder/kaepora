## Introduction

Avec la popularité grandissante de la communauté *Ocarina of Time Randomizer* (OoTR), il est de plus en plus difficile de déterminer le véritable niveau de chaque joueur. De plus, un nombre toujours plus important de joueurs exprime un souhait de voir un nouveau format voir le jour au sein de la communauté.

Certaines communautés rando ont déjà répondu à cette demande, avec notamment la communauté *A Link To The Past Randomizer*.

Voyant cela, nous avons décidé de mettre en place nous aussi une nouvelle manière de profiter de tout ce que le randomizer a à offrir.
Nous sommes heureux de vous présenter notre système de race ladder OoTR.

### Comment cela fonctionne ?

Le système de classement utilisé pour le ladder OoTR est un classement *Glicko-2*.
De ce fait, l’évolution du score de chacun s’effectue par le biais de matchs en un contre un. Des points seront ajoutés ou retirés au joueur en fonction de ses performances de telle sorte que le nombre de points des deux participants, supposés de niveau égal, se rapproche de l’égalité.

Le nombre de points gagnés ou perdus par le joueur dépendra en partie de l'écart de classement entre son adversaire et lui, ainsi que les performances passées de chacun.

Aucun joueur ne connaît son adversaire. L’identité des deux membres du binôme ainsi que leurs temps respectifs ne sera révélée que lorsque chacun d’entre-eux aura terminé sa seed.

À chaque fin de session, le classement sera mis à jour en fonction des résultats de chacun.

### Qui peut rejoindre les courses ladder OoTR ?

Tout joueur souhaitant progresser dans *Ocarina of Time Randomizer* est libre de rejoindre les différentes courses qui sont programmées.

Nous tenons à garder cette notion importante qu’est l’accessibilité. Peu importe vos performances, vous aurez un adversaire à votre niveau pour vous permettre de donner le meilleur de vous-même tout en évitant cette sensation de déséquilibre notable que peuvent donner certaines races classiques.

## Aspect technique

### Les ligues OoTR Ladder

Chacune des ligues qui composent le ladder OoTR se base sur le ruleset **Standard**. Si vous souhaitez plus d’informations sur l’ensemble des tricks autorisés par le ruleset, nous vous invitons à consulter [cette page (en anglais)](https://wiki.ootrandomizer.com/index.php?title=Standard).

Chaque ligue aura un thème qui lui est propre. À l’heure actuelle seule la ligue Standard est disponible.
Il s’agira de la ligue la plus représentative pour tout joueur souhaitant connaître son véritable niveau. Cette ligue proposera uniquement des seeds dans des **Weekly Settings**, settings qui vont continuer à évoluer au grès de la communauté.

### Déroulement d'une course ladder OoTR

#### Pré-requis

**Toute inscription aux races s’effectue sur le serveur Discord OoTR Ladder.** Vous pouvez le rejoindre via ce lien : [https://discord.gg/RCFqcMF](https://discord.gg/RCFqcMF)

À votre arrivée sur le serveur, il vous sera demandé de lancer quelques commande avant de pouvoir rejoindre une race. La commande `!help` du bot développé à cet effet vous donnera tous les outils nécessaires.

Voici une liste exhaustive des commandes du bot Discord vous permettant de remplir tous les pré-requis pour rejoindre une race :

- `!help` : Affiche un message regroupant l’ensemble des commandes
- `!leagues` : Affiche la liste des ligues proposées sur OoTR Ladder
- `!leaderboard <Ligue>` : Affiche le top 20 de la ligue souhaitée
- `!recap <Ligue>` : Affiche les résultats des 1v1 de la session en cours
- `!register` : Donne la possibilité au membre de rejoindre les races
- `!register <Pseudo>` : Donne la possibilité au membre de rejoindre les races avec un pseudo particulier
- `!rename <Pseudo>` : Permet de changer son pseudo sur OoTR

#### Rejoindre une room

Lorsque vous vous êtes inscrit pour participer à votre première race avec la commande `!register`, vous allez pouvoir guetter les annonces du bot Discord pour la ligue qui vous intéresse.
Chaque ligue a son propre canal d’annonce.

Chaque race est matérialisée sous la forme de room. Une room s’ouvrira toujours 1 heure avant le début de la race. Pour la rejoindre, il vous suffit de taper la commande indiquée par l’annonce.

Si vous avez rejoint une room mais qu’un imprévu se manifeste vous empêchant de garantir votre présence, vous pouvez toujours annuler votre participation ***jusqu’à T-15 minutes*** avec la commande `!cancel`. Aucune pénalité ne vous sera attribuée.

Lorsqu’il ne restera plus que **15 minutes** avant le départ des différents matchs, la room sera verrouillée : il sera impossible d’annuler sa participation ou même de rejoindre la room.
Toutes les personnes inscrites se verront désigner un adversaire, et la seed correspondant à votre match 1v1 vous sera envoyée via message privé Discord. Vous aurez alors 15 minutes pour vous préparer.

Vous l’aurez compris, **chaque match 1v1 constituant la room aura une seed qui lui est propre** : seul votre adversaire se verra attribuer la même seed Ocarina of Time Randomizer que la vôtre.

Un dernier rappel vous sera envoyé une minute avant le début de votre match, deux autres rappels à 30 secondes ainsi que 10 secondes. Un décompte se lancera dans les 5 dernières secondes avant que le bot donne le top départ à l’intégralité des joueurs inscrits.

<div class="notification is-danger is-light">
<strong>Dans le cas où un nombre de joueurs impair est inscrit dans la room</strong> … c’est la dure loi du « premiers arrivés, premiers servis » qui s’applique. Comme il est tout simplement impossible de jouer contre soi-même, la participation de la  dernière personne s’étant inscrite dans la room sera automatiquement annulée. Naturellement, aucune pénalité ne sera attribuée à ce joueur.
</div>

Voici une liste exhaustive des commandes du bot Discord qui vous seront utiles pour la préparation de vos matchs 1v1 OoTR Ladder :

- `!join <Ligue>` : Rejoindre une room ouverte de la ligue souhaitée
- `!cancel` : Annuler votre participation (commande inutilisable dans les 15 dernières minutes précédant le début de votre match)

#### Départ des matchs

Le bot annonce le top départ. À partir de là, nous ne pouvons que vous souhaiter bonne chance et de **respecter scrupuleusement** l’ensemble des règles OoTR Ladder en vigueur que vous pouvez consulter [ici](../rules).

Nous encourageons tous les participants de chaque match à terminer leurs seeds et à abandonner le moins souvent possible. Ne pas vous donner l’identité de votre adversaire est un choix que nous pensons compatible avec cette philosophie.

Lorsque le coup final a été porté, il vous suffira d’utiliser la commande `!done` en message privé au bot Kaepora. Dans le cas où vous ne pouvez malheureusement pas terminer votre seed, utilisez la commande `!forfeit`.

Lorsque le dernier membre du binôme a utilisé l’une de ces deux commande, votre match 1v1 est officiellement terminé et l’identité de votre adversaire ainsi que vos temps respectifs seront dévoilés publiquement sur le serveur Discord.

**La mise à jour des points ne s’effectuera cependant que lorsque l’ensemble des matchs 1v1 constituant la session seront terminés.**

## Outils utiles pour vos courses

### Trackers

- EmoTracker : [https://emotracker.net/](https://emotracker.net/)
- LinSo Tracker : [https://pastebin.com/vYrNGweu](https://pastebin.com/vYrNGweu)
- Tracker Web : [https://track-oot.net/](https://track-oot.net/)
