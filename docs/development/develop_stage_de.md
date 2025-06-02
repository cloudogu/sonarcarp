# Entwicklungsmodus

## Konfiguration anpassen
`carp.yaml` anpassen.

## externes SonarQube im CES-CAS erlauben

Zum lokalen Testen einiger Dogus ist es notwendig, den CAS in den Entwicklungsmodus zu versetzen.
Das führt dazu, dass alle Applikationen sich über den CAS authentifizieren können, auch wenn sie dort nicht
konfiguriert sind.
Dafür muss die Stage des EcoSystems auf
`development` gesetzt werden und das Dogu neu gestartet werden:

```
etcdctl set /config/_global/stage development
docker restart cas
```

## SonarQube starten
```
docker compose up -d && docker compose logs sonar -f
```

## CAS-Login testen

1. Im Browser diese URL aufrufen: http://localhost:8080/sonar
   - bei Aufruferfolg zum CAS weitergeleitet werden, die unter diesem `carp.yaml`-Property konfiguriert wurde: `cas-url`
2. Im CAS anmelden
   - bei Anmeldeerfolg zum SonarQube weitergeleitet werden, die unter diesem `carp.yaml`-Property konfiguriert wurde: `https://localhost:9090/sonar/
cas`  


## CAS-Logout testen



Bei Misserfolg kann u. U. eine manuelle Eingabe der URL http://localhost:8080/sonar/sessions/logout helfen. Diese URL wird
in diesem `carp.yaml`-Property konfiguriert: `logout-path`

## SonarQube wieder abreißen
```
docker compose stop && docker compose rm -f
```