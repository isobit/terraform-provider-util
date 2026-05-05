environment: string @tag(env)

server: {
	host: "0.0.0.0"
	port: 8080
}

replicas: int @tag(replicas, type=int)
