# VHS tape for the prr demo GIF. Render with: sh docs/demo/render.sh
# (render.sh builds prr and puts a fake "claude" on PATH so this is reproducible
# and never calls a real agent.)
Output docs/demo.gif

Require prr

Set Shell "bash"
Set FontSize 20
Set Width 1200
Set Height 760
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 60ms

Sleep 600ms
Type `prr "add dark mode to the settings page"`
Sleep 700ms
Enter
Sleep 2.8s

Type "follow the OS theme, and persist the choice"
Sleep 700ms
Enter
Sleep 3s

Type "y"
Sleep 400ms
Enter
Sleep 2.8s
