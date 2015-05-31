# tweetsbot
A data fetcher to get everything about a topic from public tweet stream

## Setup
1. Power on the server

2. The tweetbot will be automatically started
  
3. Start the web server 
   nohup go run web.go > ./log.txt &
   
4. forward request from port 80 to port 8080
   sudo iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port 8080
