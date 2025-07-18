# Guitare Hetic - Routeur eHub vers Art-Net

Ce projet est une implémentation logicielle développée dans le cadre du cours "Glassworks Course LED". Il répond à l'objectif principal (P1) : développer un module de routage performant et configurable pour les installations lumineuses du Groupe LAPS.

Ce logiciel, nommé **Guitare Hetic**, sert de pont entre l'outil de création artistique "Tan" (qui émet sur le protocole propriétaire **eHub**) et les contrôleurs LED physiques (qui reçoivent des commandes via le protocole **Art-Net**). Il est conçu pour gérer des installations complexes comptant des milliers de LEDs, comme le mur LED de test (16 384+ LEDs).

## Table des matières
- [Fonctionnalités](#fonctionnalités)
- [Aperçu de l'application](#aperçu-de-lapplication)
- [Architecture logicielle](#architecture-logicielle)
- [Technologies utilisées](#technologies-utilisées)
- [Guide d'utilisation](#guide-dutilisation)
- [Configuration](#configuration)
- [Auteurs](#auteurs)

## Fonctionnalités

Le projet implémente l'ensemble des exigences du cahier des charges, y compris des fonctionnalités bonus.

-   **E1 : Réception du protocole eHub** : Le logiciel écoute sur un port UDP configurable (par défaut 8765) et est capable de parser et d'interpréter les messages `config` et `update` du protocole eHub.
-   **E2 & E5 : Moniteurs en temps réel** : Une interface graphique permet de visualiser en direct, pour un univers DMX donné, les données de couleur reçues depuis eHub et les données DMX correspondantes envoyées en Art-Net.
-   **E3 : Correspondance flexible via Excel** : La configuration du routage (correspondance entre les entités logiques et les sorties physiques des contrôleurs) se fait via un simple fichier Excel ou CSV, ce qui rend l'édition pour de très grandes installations rapide et intuitive.
-   **E4 : Routage haute performance** : Le cœur du routage est écrit en Go et optimisé pour une faible latence, une utilisation minimale du CPU et de la mémoire. Il est capable de gérer des milliers de mises à jour par seconde.
-   **E6 : Sauvegarde et chargement** : L'interface permet de charger un fichier de configuration de routage au démarrage et de sauvegarder la configuration actuelle (par exemple après avoir modifié des adresses IP) dans un nouveau fichier Excel.
-   **E7 : Contrôle de la charge réseau** : L'envoi des paquets Art-Net est cadencé (par défaut à 30 FPS) pour éviter la saturation du réseau, tout en garantissant une fluidité visuelle parfaite.
-   **E8 : Patching en direct** : Une fonctionnalité de "patching" permet de ré-acheminer des circuits DMX à la volée en chargeant un simple fichier Excel. C'est idéal pour contourner une panne matérielle sur le terrain sans avoir à reconfigurer toute l'installation.
-   **E10 (Bonus) : Simulateur eHub ("Faker")** : Un module de test est intégré pour générer des patterns de couleur (fixes ou animées) et les envoyer dans le routeur. Cela permet de tester la configuration de routage et l'installation physique sans avoir besoin de lancer Unity.

## Aperçu de l'application

L'application fournit une interface graphique claire pour piloter et surveiller le routage.

| Vue principale (Liste des contrôleurs) | Vue détaillée (Univers d'un contrôleur) | Vue Moniteur (Entrées/Sorties d'un univers) |
| :---: | :---: | :---: |
| _Liste des contrôleurs détectés depuis le fichier de configuration._ | _Détail des univers et des plages d'entités pour un contrôleur sélectionné._ | _Visualisation en temps réel des couleurs eHub (gauche) et Art-Net (droite)._ |

## Architecture logicielle

Le projet suit une architecture "Clean Architecture" pour garantir la séparation des responsabilités, la testabilité et la maintenabilité.

-   **`internal/domain`** : Contient les objets et la logique métier purs (définitions des paquets eHub, Art-Net, etc.).
-   **`internal/application`** : Orchestre les cas d'usage. Le `processor/service.go` est le cœur qui reçoit les données eHub, les traite selon la configuration et prépare les données DMX.
-   **`internal/infrastructure`** : Gère les aspects techniques externes : écoute réseau (`ehub/listener.go`), envoi des paquets Art-Net (`artnet/sender.go`).
-   **`internal/ui`** : Contient toute la logique de l'interface graphique développée avec Fyne (vues, contrôleur UI, état).
-   **`internal/config`** : Gère le chargement et la sauvegarde des configurations depuis/vers des fichiers (Excel, CSV).
-   **`internal/simulator`** : Implémente le "Faker" eHub pour les tests.

Le flux de données est le suivant :
`Listener eHub` -> `Parser eHub` -> `Service de traitement (Routage + Patching)` -> `File d'attente Art-Net` -> `Sender Art-Net`

## Technologies utilisées

-   **Langage** : [Go (Golang)](https://golang.org/)
-   **Interface Graphique** : [Fyne](https://fyne.io/) - un toolkit cross-platform en Go.
-   **Lecture de fichiers Excel** : [Excelize](https://github.com/qax-os/excelize)

## Guide d'utilisation

### Prérequis

-   Avoir Go installé sur votre machine.

### Compilation et exécution

1.  Clonez le dépôt.
2.  Ouvrez un terminal à la racine du projet.
3.  Compilez l'application :
    ```bash
    go build -o GuitareHetic
    ```
4.  Exécutez l'application :
    ```bash
    ./GuitareHetic
    ```

### Workflow

1.  Lancez l'application.
2.  Via le menu `Art'hetic` > `Charger configuration...`, sélectionnez le fichier Excel ou CSV qui décrit votre installation (par exemple `internal/config/routing.csv`).
3.  L'interface affiche la liste des contrôleurs (par adresse IP).
4.  Cliquez sur une IP pour voir les univers qu'elle gère.
5.  Cliquez sur le bouton `Monitorer` d'un univers pour visualiser le flux de données en temps réel.
6.  Utilisez le menu `Faker` pour envoyer des données de test à l'installation.
7.  Utilisez le menu `Patching` pour charger un fichier de patch et l'activer/désactiver.

## Configuration

### Fichier de routage (`.xlsx` ou `.csv`)

Le fichier de routage principal doit contenir les colonnes suivantes :

| Name | Entity Start | Entity End | ArtNet IP | ArtNet Universe |
| :--- | :--- | :--- | :--- | :--- |
| Strip 1 | 100 | 269 | 192.168.1.45 | 0 |
| Strip 2 | 270 | 358 | 192.168.1.45 | 1 |
| ... | ... | ... | ... | ... |

### Fichier de Patch (`.xlsx`)

Le fichier de patching permet de rediriger un canal DMX vers un ou plusieurs autres. Il doit contenir 3 colonnes : `Universe`, `SourceChannel`, `DestinationChannel`.

| Universe | SourceChannel | DestinationChannel |
| :--- | :--- | :--- |
| 5 | 1 | 389 |
| 5 | 2 | 390 |
| ... | ... | ... |

Dans cet exemple, pour l'univers 5, les données destinées au canal 1 seront envoyées au canal 389, et celles du canal 2 au canal 390.

## Auteurs

*   Quimbre Adrien
*   Drici Adam
*   Travers Ayline