; Language-only program (no window)

Print("BitShin BASIC console")

Function Square(n)
    Return n * n
End Function

For i = 1 To 5
    Print(i, "squared is", Square(i))
Next

If 2 + 2 = 4 Then Print("math works")
End
