
numElements = int(raw_input("Enter the number of elements in the difference engine: "))
elements = []
for i in range(0,numElements):
    elements.append( int(raw_input("Enter next value: ")))
iterations = int(raw_input("Enter the number of iterations: "))
iterOutput = ""
for i in range(0,numElements):
    iterOutput+= str(elements[i]) + " "

print(iterOutput)

for i in range(0,iterations):
    iterOutput = ""
    for i in range(0,numElements-1):
        elements[i] = elements[i] + elements[i+1]
        iterOutput+= str(elements[i]) + " "
    iterOutput+= str(elements[numElements-1])
    print(iterOutput)
    #print(str(elements[0]))
