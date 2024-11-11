## Async Display

### Principe
- On print les stdout et stderr de chaque test dans des buffers indépendants
- On flush tous les printers d'une suite, sequentielement, dans un même fichier (deux fichiers pour stdout & stderr)
- Un daemon réalise le flush en continue en arriere plan
- Pour display en live les resultats, on tail les fichiers en cours d'écriture jusqu'à fermeture de la suite

testPrinter => buffer => suiteOutputsSink (tmp file)
suiteOutputsSink (tmp file) ==tail==> displayOutputs

- => choix du suiteOutputsSink à la création du suitePrinter par le asyncDisplay
- => tails du suiteOutputsSink par le asyncDisplay

### struct suitePrinter 
- permet d'obtenir les printers de chaque test d'une suite
- tous les printer sont bufferisés
- flusher le suitePrinter permet de pousser dans l'ordre tout ce qui a été print dans 2 fichiers out et err


### struct asyncPrinter
- Permet d'obtenir le printer associé à un test (dans une suite)
- Permet de flush tout les printers d'une suite soit une seule fois, soit en boucle jusqu'a ce que la suite soit fermée. 





### struct asyncDisplay



### Todo
- Passer le dossier tmp qui sert aux buffers à asyncDisplay.New() => eliminer la dépendance au pkg repo.
- Améliorer le fonctionnement suivant : "newAsyncPrinters outW et errW must be nil to write into tmp files"

### Generic async display ?
- plusieurs process // veulent ecrire
- gerer plusieurs "screen"
- gerer plusieurs "sortie" par screen (typiquement stdout et stderr)
- organiser les sorties en session dans un screen
- attendre la fermeture d'une session pour passer à la suivante (print sequentiel)
- ordonnancer les sorties des process au sein d'une session
- utilise des fichiers temporaires pour stoquer les sorties
- permet d'obtenir des Writer pour y ecrire
