graph LR

    subgraph Client
        direction TB
        C[Client]
        P[Parser]
    end

    subgraph Display
        S(Screen)
        A(AsyncScreen)
    end

    subgraph Context
        F[Facade]
        R[(Repo)]
        
    end

    F --> Run
    F --> Sh

    D[Daemon]

    subgraph Exec
        Run[Runner]
        Sh[Shell]
        Pod[Podman]
        Doc[Docker]
        Run --> Pod
        Run --> Doc
    end
    
    C -->|launch process| D
    C -->|check| P
    C --> F
    C -->|sync| S
    C -->|sync| A
    D -->|async| A
    A --> S
    D --> F
    F -->|SQL| R
