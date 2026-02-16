sudo pkill -f spire-agent
  sudo rm -rf /opt/spire/data/agent/*
  sudo rm -rf /run/spire/sockets/*


1. ./find-pi.sh - run this to find the find-pi

  2. Copy to Pi:
  # Binary + scripts
  scp satellite satellite.sh setup-sat-pi.sh sat-1@10.229.209.144:~/

copy the certs
scp deploy/quickstart/spiffe/x509pop/external/gc/certs/{ca.crt,agent-satellite.crt,agent-satellite.key} sat-1@10.229.209.144:~/certs/

if any errors come because of stale certs
sudo rm -rf /opt/spire/data/agent/*

  # Certs from the GC setup
  scp -r deploy/quickstart/spiffe/x509pop/external/gc/certs sat-1@10.229.209.144:~/

  3. On the Pi:
  # Install SPIRE agent + start it (downloads arm64 binary, copies certs, starts agent)
  ./setup-sat-pi.sh

  # Run satellite
  ./satellite.sh

---

  1. Laptop: cd deploy/quickstart/spiffe/x509pop/external/gc && ./cleanup.sh then cd ~/code/harbor-satellite && ./run-gc.sh
  2. Laptop: scp deploy/quickstart/spiffe/x509pop/external/gc/certs/{ca.crt,agent-satellite.crt,agent-satellite.key} sat-1@10.229.209.144:~/certs/
  3. Pi: ./setup-sat-pi.sh
  4. Laptop: ./register-sat.sh
  5. Pi: ./satellite.sh


  scp satellite satellite.sh setup-sat-pi.sh cleanup-sat-pi.sh bw-limit.sh pod.yaml sat-1@10.229.209.144:~/ && ssh sat-1@10.229.209.144 "mkdir -p ~/certs" && scp
  deploy/quickstart/spiffe/x509pop/external/gc/certs/{ca.crt,agent-satellite.crt,agent-satellite.key} sat-1@10.229.209.144:~/certs/

1. Laptop: cd deploy/quickstart/spiffe/x509pop/external/gc && ./cleanup.sh then cd ~/code/harbor-satellite && ./run-gc.sh
  2. Laptop: scp deploy/quickstart/spiffe/x509pop/external/gc/certs/{ca.crt,agent-satellite.crt,agent-satellite.key} sat-1@10.229.209.144:~/certs/
  3. Pi: ./setup-sat-pi.sh
  4. Laptop: ./register-sat.sh
  5. Pi: ./satellite.sh

