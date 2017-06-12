p c.o 1
p 1 a13.alpha
p 1 a14.alpha
p 1 a15.alpha
p 1 a16.alpha
p 2 a14.beta
p 3 a15.beta
p 4 a16.beta
p a13.A 2
p a14.A 3
p a15.A 4
p a16.A 5
# Wire all variables to output
#p a16.A 2 
#p 2 a15.a
#p a15.A 3
#p 3 a14.a
#p a14.A 4
#p 4 a13.a
#
# i.io initating unit io
p i.io 1-1
p 1-1 p.Ci
p p.C1o 1-4

p 1-4 a13.6i
p 1-4 a14.6i
p 1-4 a15.6i
p 1-4 a16.6i

s a13.op6 alpha
s a13.op6 A
s a14.op6 alpha
s a13.op6 A
s a15.op6 alpha
s a15.op6 A
s a16.op6 alpha

s p.d17s1 9
s p.d16s1 9
s p.d15s1 9
s p.d14s1 9
s p.d14s2 1
s p.cC 4

s pr.2 P
s pr.3 P
s pr.15 P
s pr.16 P
