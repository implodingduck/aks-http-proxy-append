NAME = akshttpproxyappend

app: deps  
	go build -v -o $(NAME) main.go

deps:
	go mod tidy
