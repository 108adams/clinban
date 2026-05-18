# Console system for kanban board ticket management

The goal of this project - to deliver a program called "Clinban" which allows managing the kanban board in console/terminal window.

The tickets for kanban board should be files in markdown with yaml header, they should use wiki capabilities of inter-connecting. Each ticket should track its status, tags and all the proper metadata needed to manage flow of the work. 

## technology
Golang in the newest version and its environment and packages ecosystm.

There is no database other than the files - all the project needs to be a git repo, trackable and versionable. The Clinban should treat given directory as the repository of work and properly manage the files as the part of the process. 

Integration with git is not needed - versioning will be handled independently.

## target architectures
Linux and MacOS

## Users

### Humans
The project is aimed at single develoer or a very small team developing an IT product. 
The code usually is stored in a repository. I assume we can use the same repository for tickets.

### Automata
Due to the open format of tickets I assume other actors - like CI/CD pipelines, testing infrastructure, LLM and AI developing tools and agents will be using the system extensively to track the work. Hence the shema for ticket document must be well prepared.

