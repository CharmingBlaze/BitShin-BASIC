; Struct / Type, Method, Import, Namespace — console only

Import "struct_lib.bb" As Lib

Struct Vec
    Field x
    Field y
    Method Length()
        Return Sqr(Self.x * Self.x + Self.y * Self.y)
    End Method
    Method Add(n)
        Self.x = Self.x + n
        Self.y = Self.y + n
    End Method
End Struct

Type Point
    Field x
    Field y
End Type

Namespace Util
    Function Sum(a, b)
        Return a + b
    End Function
End Namespace

v = Vec(3, 4)
Print("len", Int(v.Length()))
Print("x", v.x, "y", v.y)

w = v
w.x = 0
Print("copy x", v.x, w.x)

v.Add(1)
Print("mut", v.x, v.y)

p = New Point
p.x = 10
p.y = 20
Print("point", p.x, p.y)

Print("import", Lib.Double(5))
Print("ns", Util.Sum(2, 3))
End
