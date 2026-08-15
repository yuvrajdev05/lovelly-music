# Added community features

This build keeps the existing Go music-bot architecture and adds independent Go implementations for the requested community tools.

## Moderation
- /ban, /sban, /tban, /unban
- /kick, /skick
- /mute, /tmute, /unmute
- /warn, /swarn, /warns, /rmwarns
- /del, /purge
- /pin, /unpin, /unpinall
- /promote, /fullpromote, /demote
- /report
- /zombies
- /banall CONFIRM

## Sudo / block-chat
- /block, /unblock, /blocked
- /gban, /ungban, /gbanlist
- /blchat, /unblchat, /blchats

## Media utilities
- /tgm
- /tts <text>

## Fun / games
- /dice, /ludo, /dart, /basket, /basketball, /football, /bowling
- /truth, /dare
- /bored
- /gali (clean playful roast only)

## Virtual gifts
- /balance, /gifts, /sendgift, /mygifts, /top

## Existing features retained
TagAll, admin tagging, wish tagging, welcome messages, VC logger, user info, /play command auto-delete, and the existing music/voice-chat system remain in the project.

## Important build note
The source was syntax-parsed with Go's standard parser and the locale YAML was parsed successfully. A complete dependency build could not be executed in this environment because the project requests Go 1.25.7 while the available local toolchain is Go 1.23.2 and outbound module downloads are unavailable.

The gambling-style / commands from the Python project were intentionally not ported.
