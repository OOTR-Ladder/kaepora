## Que sont les matches de Ladder ?
Un match de Ladder est une course en un contre un où vous ne connaissez pas
votre adversaire avant d'avoir terminé. Le but est de finir une partie de
_Ocarina of Time: Randomizer_ (OoTR) le plus rapidement possible.  
Quand tous les matches d'une sessions sont terminés, les scores sont mis à
jour et les trois meilleurs joueurs ont l'honneur d'apparaître sur le podium.

## S'inscrire et participer
Pour participer vous devez [lire les règles](/rules) et vous inscrire en
envoyant la commande `!register` dans le canal `#ladder-signup` du [serveur
Discord OoTR](https://discord.gg/yZtdURz). Le bot _Kaepora_ vous répondra en
message privé et c'est dans ces messages privés que vous devrez taper toutes
les autres commandes.

<div class="message is-warning">
    <div class="message-body">
        <p>Vous devez <em>autoriser les messages privés en provenance des
        members du serveur</em> dans vos paramètres Discord de
        <em>Confidentialité & Sécurité</em> sinon le bot ne pourra pas vous
        contacter.</p>
    </div>
</div>

Quand une session approche, _Kaepora_ fera une annonce sur un canal dédié dans
la catégorie _IR Ladder_ du serveur. Après cette annonce, vous pourrez envoyer
à _Kaepora_ la commande `!join` suivie du nom court de la ligue dans laquelle
vous voulez participer. Par exemple pour la ligue _Standard_ vous devrez
envoyer `!join std`.  
Si vous voulez annuler votre inscription envoyez `!cancel`, votre score ne sera
pas affecté.

<div class="message is-warning">
    <div class="message-body">
        <p>S'il y a un nombre impair de joueurs, la dernière personne a avoir
        rejoint la session sera exclue.</p>
    </div>
</div>

**Quinze minutes avant que le match ne commence** _Kaepora_ vous enverra un lien
vers votre _seed_ sur [ootrandomizer.com](https://ootrandomizer.com), vous
pourrez y patcher votre ROM et l'installer sur votre Wii ou votre émulateur
([Retroarch](https://www.retroarch.com/) avec le noyeau ParaLLEl est fortement
recommandé, Bizhawk et Project64 1.6-1.7 sont aussi autorisés).

Le match commence après un compte à rebours donné par _Kaepora_ sur le canal de
la ligue que vous avez rejoint (par exemple `#ladder-league-standard`).  
Tous les matches commencent au même moment, si vous êtes en retard le match ne
sera pas reporté et vous partirez avec un désavantage.

Une fois votre partie terminée (soit en portant le coup de grâce à Ganon, soit
en trouvant la dernière pièce de Triforce dans certains matches de _Shuffled
Settings_) vous devez envoyer `!done` à _Kaepora_.
Le bot vous donnera alors votre temps, le nom de votre adversaire, son temps
s'il a fini avant vous, et votre _spoiler log_ contenant l'emplacement et
l'ordre d'obtention de tous les objets que vous aviez à trouver (cette
information est aussi disponible sur le lien de votre _seed_).

Si vous ne pouvez pas finir votre partie, peu importe la raison, vous devez
envoyer `!forfeit`. Un forfait vous fait perdre automatiquement la partie, sauf
si votre adversaire déclare aussi forfait auquel cas le match se termine par
une égalité.

<div class="message is-info">
    <div class="message-header"><p>En résumé :</p></div>
    <div class="message-body">
        <ul>
        <li><code>!help</code> contient l'aide du bot et devrait être la première commande que vous envoyez.</li>
        <li><code>!register</code> pour vous inscrire au Ladder.</li>
        <li><code>!join &lt;short name&gt;</code> quand une session commence.</li>
        <li><code>!done</code> sur le coup de grâce ou <code>!forfeit</code>.</li>
        </ul>
    </div>
</div>

## Scores
Chaque ligue a ses propres scores qui sont gérés par le [système de classement
Glicko-2][1].

Ce système a quelques particularité :

 - Votre niveau n'est pas représenté par un seul nombre mais par une plage dans
   laquelle on estime que votre niveau réel se trouve à 95 % de confiance.
 - La récompense (ou la pénalité) pour une victoire (ou une défaite) est
   proportionnelle à l'écart de niveau entre les joueurs.
 - Vous devez finir plusieurs matches pour faire baisser votre _déviation de
   niveau_ et atteindre le palier nécessaire pour apparaître sur le tableau des
   scores.
 - Ne pas faire de match pendant une semaine augmentera votre _déviation de niveau_.

[1]: https://fr.wikipedia.org/wiki/Classement_Glicko

## Matchmaking
L'algorithme de matchmaking sélectionne des joueurs avec des plages de niveau
qui se chevauchent (`R±2×RD` dans le système Glicko-2) en préférant les joueurs
ayant la plus petite différence de niveau.  
Cette méthode garanti un matchmaking _globalement_ juste mais il y a toujours
la possibilité qu'un joueur de très haut niveau soit incompatible avec tous les
autres joueurs d'une session et se retrouve contre un joueur qui sera en fort
désavantage. Cette situation est _inévitable_ à moins de retirer des joueurs de
la session.

Si vous vous retrouvez contre un adversaire bien meilleur que vous, vous pouvez
soit regarder ses vidéos pour comprendre où et comment vous améliorer, soit
venir vous plaindre de combien le matchmaking est injuste. Soyez le meilleur
joueur, _git gud_.

## Ligues
### Standard (`std`)
La ligue standard est la ligue par défaut que tous les joueurs de tous les
niveaux devraient rejoindre pour mettre à l'épreuve leur chance et leurs
compétences.  
La liste des _tricks_ et _glitches_ autorisés dans cette ligue est définie par
le [OoTR Standard Racing Ruleset (SRR)][2].

Il y a plusieurs sessions par jour et vous pouvez en rejoindre autant que vous
voulez.

Chaque versus a sa propre _seed_.

[2]: https://wiki.ootrandomizer.com/index.php?title=Standard

### Shuffled Settings (`shu`)
La ligue _Shuffled Settings_ ou « paramètres mélangés » utilise une génération
de _seed_ différente où les paramètres du générateur lui-même sont choisis
aléatoirement.

Les paramètres ne sont pas choisis complètement au hasard : l'algorithme itère
sur une liste _pondérée_ de valeurs pour en choisir une aléatoirement et ajoute
son _coût_ au décompte provisoire.  
Quand le décompte provisoire atteint le coût budgétisé, l'algorithme s'arrête
et applique les paramètres choisis par dessus les paramètres de la ligue
standard.

Les poids et probabilités ont été choisis pour permettre des _seeds_
intéressantes sans pour autant forcer les joueurs à faire des parties
interminables en _fullsanity_.

Chaque versus a ses propres paramètres et sa propre _seed_.

## Contribuer
Le Ladder est un effort collaboratif diffusé sous la licence libre MIT.  
Vous pouvez contribuer en envoyant une _pull request_ au dépôt [Kaepora][3] qui
contient les sources du bot et du site web.

[3]: https://github.com/OOTR-Ladder/kaepora
