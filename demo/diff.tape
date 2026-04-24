# VHS tape: showcase `bt pr diff` and `bt compare`.
# Run with: vhs demo/diff.tape
# Output:    demo/diff.gif

Output demo/diff.gif

Set Shell zsh
Set FontSize 16
Set Width 1200
Set Height 700
Set Theme "Tokyo Night"
Set TypingSpeed 45ms

Type "bt pr list --repo myorg/service --state open"
Enter
Sleep 2s

Type "bt pr diff 142"
Enter
Sleep 3s

Type "q"
Sleep 500ms

Type "bt compare main...feat/kafka-38 --stat"
Enter
Sleep 3s

Type "bt compare main...feat/kafka-38 --commits"
Enter
Sleep 3s
