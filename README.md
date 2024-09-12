# P2P File Sharing System

This is a peer-to-peer (P2P) file-sharing system built using gRPC for communication. Peers can request or upload files, and a tracker service helps peers locate files across the network.

## Features
- Peers can upload files to the network.
- Peers can request files from other peers.
- A central tracker maintains a registry of peers and their files.
- Peers communicate with each other and the tracker using gRPC.

### Running The System

- **Pull Docker Images**:
   ```bash
   make docker-pull
- **After ensuring that Docker Compose is set up, you can bring up the system with**:
   ```bash
   make up 
- **To take down the system:**:
   ```bash
   make down 
- **To view logs:**:
   ```bash
   make logs 

## Operations

1. **File Upload:**
Ensure you have created a static directory in the client and placed your file (e.g., file.txt) in it.
   ```bash
   make upload
   ```

2. **File Upload:**
Ensure you have created a static directory in the client and placed your file (e.g., file.txt) in it.
   ```bash
   make upload
   ```
3. **Get Registered Peers:**
    ```bash
    make get-peers
    ``` 


