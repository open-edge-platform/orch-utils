# Example graph to parse

```
                  Root
                   |
                   |
                  Config
                 /     \
                /       \
    softlink  /         \
 Dns------->Gns         AccessControlPolicy
             |                  |
             |                  |
         SvcGroup ----------> ACPConfig
                   softlink

```

DSL example based on https://confluence.eng.vmware.com/pages/viewpage.action?spaceKey=NSBU&title=Nexus+Platform#NexusPlatform-TL;DR;
