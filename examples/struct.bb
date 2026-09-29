; Struct & Type — methods, Import, Namespace, and shared handles (console)

Import "struct_lib.bb" As Lib

; Struct with methods (Length / Add)
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

; Classic Type (fields only)
Type Point
    Field x
    Field y
End Type

; Namespace-qualified function
Namespace Util
    Function Sum(a, b)
        Return a + b
    End Function
End Namespace

; Construct and call methods
v = Vec(3, 4)
Print("len", Int(v.Length()))
Print("x", v.x, "y", v.y)

; Assignment shares the same handle (v and w are one object)
w = v
w.x = 0
Print("shared x", v.x, w.x)

v.Add(1)
Print("mut", v.x, v.y)

; New Point instance
p = New Point
p.x = 10
p.y = 20
Print("point", p.x, p.y)

; Imported Lib.Double and Util.Sum
Print("import", Lib.Double(5))
Print("ns", Util.Sum(2, 3))
End
