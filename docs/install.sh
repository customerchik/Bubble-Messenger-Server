git clone https://github.com/customerchik/Bubble-Messenger-Server Main
git clone https://github.com/mongolianpes/Bubble-Messenger-Server-Users Users
git clone https://github.com/mongolianpes/Bubble-Messenger-Server-Messenger Messenger
git clone https://github.com/mongolianpes/Bubble-Messenger-Server-Files Files
git clone https://github.com/mongolianpes/Bubble-Messenger-Server-AudioDialog AudioDialog

cp Main/docs/docker-compose.yml .
cp -r Main/docs/DB .

docker compose up -d