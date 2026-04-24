# VHS tape: 30-second tour of bt.
# Run with: vhs demo/tour.tape

Output demo/tour.gif

Set Shell zsh
Set FontSize 16
Set Width 1200
Set Height 700
Set Theme "Tokyo Night"
Set TypingSpeed 45ms

Type "bt --version"
Enter
Sleep 1s

Type "bt context list"
Enter
Sleep 2s

Type "bt repo list --workspace myorg | head"
Enter
Sleep 2s

Type "bt pr list --repo myorg/service --state open"
Enter
Sleep 2s

Type "bt pr view 142"
Enter
Sleep 3s
