#!/usr/bin/env python3
"""
ChromaDB Server Management Script
Usage:
  python chroma_manager.py status
  python chroma_manager.py start [--host HOST] [--port PORT] [--path PATH]
  python chroma_manager.py stop
  python chroma_manager.py restart
"""

import argparse
import os
import signal
import subprocess
import time
from pathlib import Path

import chromadb


def check_server_status(host: str = "localhost", port: int = 8000) -> bool:
    """Check if ChromaDB server is running"""
    # noinspection PyBroadException
    try:
        client = chromadb.HttpClient(host=host, port=port)
        response = client.heartbeat()
        return True
    except Exception:
        return False


def start_server(host: str = "localhost", port: int = 8000, db_path: str = "./chroma_db", background: bool = True):
    """Start ChromaDB server"""
    if check_server_status(host, port):
        print(f"✓ ChromaDB server is already running at {host}:{port}")
        return True

    print(f"Starting ChromaDB server at {host}:{port}...")
    print(f"Database path: {db_path}")

    # Ensure database directory exists
    Path(db_path).mkdir(parents=True, exist_ok=True)

    # Use uvx to run ChromaDB server
    cmd = [
        "uvx", "--from", "chromadb", "chroma", "run",
        "--host", host,
        "--port", str(port),
        "--path", db_path
    ]

    try:
        if background:
            # Start in background
            process = subprocess.Popen(
                cmd,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                start_new_session=True
            )

            # Wait a moment and check if it started successfully
            time.sleep(2)
            if check_server_status(host, port):
                print(f"✓ ChromaDB server started successfully (PID: {process.pid})")

                # Save PID for later stopping
                pid_file = f".chroma_server_{port}.pid"
                with open(pid_file, 'w') as f:
                    f.write(str(process.pid))

                return True
            else:
                print("✗ Failed to start ChromaDB server")
                return False
        else:
            # Start in foreground
            subprocess.run(cmd, check=True)
            return True

    except subprocess.CalledProcessError as e:
        print(f"✗ Failed to start ChromaDB server: {e}")
        return False
    except FileNotFoundError:
        print("✗ uvx or ChromaDB not found.")
        print("Install uv with: curl -LsSf https://astral.sh/uv/install.sh | sh")
        print("Or try: pip install chromadb")
        return False


def stop_server(port: int = 8000):
    """Stop ChromaDB server"""
    pid_file = f".chroma_server_{port}.pid"

    if os.path.exists(pid_file):
        try:
            with open(pid_file, 'r') as f:
                pid = int(f.read().strip())

            os.kill(pid, signal.SIGTERM)
            time.sleep(1)

            # Check if process is really stopped
            try:
                os.kill(pid, 0)  # Check if process exists
                print(f"Force killing ChromaDB server (PID: {pid})")
                os.kill(pid, signal.SIGKILL)
            except ProcessLookupError:
                pass  # Process already stopped

            os.unlink(pid_file)
            print(f"✓ ChromaDB server stopped (PID: {pid})")
            return True

        except Exception as e:
            print(f"✗ Failed to stop server using PID file: {e}")

    # Try to find and kill chroma processes
    try:
        result = subprocess.run(
            ["pgrep", "-f", f"chroma.*run.*{port}"],
            capture_output=True,
            text=True
        )

        if result.returncode == 0:
            pids = result.stdout.strip().split('\n')
            for pid in pids:
                if pid:
                    os.kill(int(pid), signal.SIGTERM)
                    print(f"✓ Stopped ChromaDB process (PID: {pid})")
            return True
        else:
            print("✓ No ChromaDB server processes found")
            return True

    except Exception as e:
        print(f"✗ Failed to stop ChromaDB server: {e}")
        return False


def show_status(host: str = "localhost", port: int = 8000):
    """Show server status"""
    if check_server_status(host, port):
        print(f"✓ ChromaDB server is running at {host}:{port}")

        try:
            client = chromadb.HttpClient(host=host, port=port)
            collections = client.list_collections()
            print(f"  Collections: {len(collections)}")
            for collection in collections:
                count = collection.count()
                print(f"    - {collection.name}: {count} documents")
        except Exception as e:
            print(f"  Could not get collection info: {e}")
    else:
        print(f"✗ ChromaDB server is not running at {host}:{port}")


def main():
    parser = argparse.ArgumentParser(description="Manage ChromaDB server")
    parser.add_argument("command", choices=["start", "stop", "restart", "status"], help="Command to execute")
    parser.add_argument("--host", default="localhost", help="Server host (default: localhost)")
    parser.add_argument("--port", type=int, default=8000, help="Server port (default: 8000)")
    parser.add_argument("--path", default="./chroma_db", help="Database path (default: ./chroma_db)")
    parser.add_argument("--foreground", action="store_true", help="Run server in foreground")

    args = parser.parse_args()

    if args.command == "status":
        show_status(args.host, args.port)

    elif args.command == "start":
        start_server(args.host, args.port, args.path, background=not args.foreground)

    elif args.command == "stop":
        stop_server(args.port)

    elif args.command == "restart":
        print("Restarting ChromaDB server...")
        stop_server(args.port)
        time.sleep(2)
        start_server(args.host, args.port, args.path, background=not args.foreground)


if __name__ == "__main__":
    main()