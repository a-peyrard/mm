import json
import subprocess
import tempfile
import time
import uuid
from typing import Dict, List, Optional

import chromadb
import pytest
from chromadb import QueryResult
from sentence_transformers import SentenceTransformer

from indexer import index_chunks


@pytest.fixture
def temp_path():
    with tempfile.TemporaryDirectory() as temp_dir:
        yield temp_dir

@pytest.fixture(scope="session")
def model() -> SentenceTransformer:
    return SentenceTransformer("all-MiniLM-L6-v2")


class IndexerDaemon:
    def __init__(self, db_path: str):
        self.db_path = db_path
        self.process = None
        self.stdin = None
        self.stdout = None

    def start(self):
        self.process = subprocess.Popen(
            ["uv", "run", "indexer.py", f"--db-path={self.db_path}"],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            bufsize=1  # Line buffered
        )
        self.stdin = self.process.stdin
        self.stdout = self.process.stdout

        self.wait_for_ready()

        if self.process.poll() is not None:
            stderr = self.process.stderr.read()
            raise Exception(f"Indexer daemon failed to start: {stderr}")

    def wait_for_ready(self, timeout: float = 10.0):
        start_time = time.time()

        while time.time() - start_time < timeout:
            if self.process.poll() is not None:
                stderr = self.process.stderr.read()
                raise Exception(f"Daemon died before sending READY: {stderr}")

            # noinspection PyBroadException
            try:
                import select
                if select.select([self.stdout], [], [], 0.1)[0]:
                    line = self.stdout.readline()
                    if line.strip():
                        try:
                            ready_msg = json.loads(line.strip())
                            if ready_msg.get("status") == "READY":
                                return
                            else:
                                raise Exception(f"Expected READY status, got: {ready_msg}")
                        except json.JSONDecodeError:
                            raise Exception(f"Expected JSON READY message, got: {line.strip()}")
            except Exception:
                time.sleep(0.1)
                continue

        raise Exception(f"Daemon did not send READY within {timeout} seconds")

    def send_request(self, chunks: List[Dict], req_id: Optional[str] = None) -> Dict:
        """Send an indexing request to the daemon."""
        if not self.process or self.process.poll() is not None:
            raise Exception("Daemon is not running")

        # Send request as JSON line
        request = {
            "meta": {
              "id": req_id or str(uuid.uuid4())
            },
            "chunks": chunks
        }
        json_line = json.dumps(request) + "\n"
        self.stdin.write(json_line)
        self.stdin.flush()

        # Read response
        response_line = self.stdout.readline()
        if not response_line:
            raise Exception("No response from daemon")

        return json.loads(response_line.strip())

    def stop(self):
        """Stop the indexer daemon."""
        if self.process and self.process.poll() is None:
            # Send exit command
            try:
                self.stdin.write("exit\n")
                self.stdin.flush()

                # Wait for graceful shutdown
                self.process.wait(timeout=5)
            except (subprocess.TimeoutExpired, BrokenPipeError):
                # Force kill if needed
                self.process.terminate()
                try:
                    self.process.wait(timeout=2)
                except subprocess.TimeoutExpired:
                    self.process.kill()

            self.process = None

    def __enter__(self):
        self.start()
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.stop()


def search(query: str, db_path: str, n_results: int = 5) -> QueryResult:
    client = chromadb.PersistentClient(path=db_path)
    collection = client.get_collection("code_chunks")

    results = collection.query(
        query_texts=[query],
        n_results=n_results
    )

    return results


@pytest.mark.slow
def describe_e2e_tests_for_index_chunks():
    def test_should_allow_to_index_and_search(temp_path):
        # GIVEN
        chunks = [
            {
                "id": "auth.js_func_validateToken_15",
                "type": "function",
                "content": "function validateToken(token) { return jwt.verify(token, secret); }",
                "metadata": {
                    "file_path": "/src/auth.js",
                    "function_name": "validateToken",
                    "language": "javascript"
                }
            },
            {
                "id": "utils.py_func_calculateTax_8",
                "type": "function",
                "content": "def calculate_tax(income): return income * 0.3 if income > 50000 else income * 0.2",
                "metadata": {
                    "file_path": "/src/utils.py",
                    "function_name": "calculate_tax",
                    "language": "python"
                }
            }
        ]

        # WHEN
        with IndexerDaemon(temp_path) as daemon:
            result = daemon.send_request(chunks)
            assert result["status"] == "success"
            assert result["indexed_count"] == 2

        # THEN
        results = search(query="authentication token validation", db_path=temp_path, n_results=1)
        assert len(results["ids"][0]) > 0
        assert "validateToken" in results["documents"][0][0]

    def test_should_return_request_id_in_response(temp_path):
        # GIVEN
        chunks = [
            {
                "id": "auth.js_func_validateToken_15",
                "type": "function",
                "content": "function validateToken(token) { return jwt.verify(token, secret); }",
                "metadata": {
                    "file_path": "/src/auth.js",
                    "function_name": "validateToken",
                    "language": "javascript"
                }
            },
        ]

        # WHEN & THEN
        with IndexerDaemon(temp_path) as daemon:
            result = daemon.send_request(chunks, req_id="test_request_123")
            assert result["id"] == "test_request_123"

    def test_should_handle_multiple_requests(temp_path):
        # GIVEN
        chunks1 = [
            {
                "id": "test1",
                "type": "function",
                "content": "function first() { return 'first'; }",
                "metadata": {"file_path": "/test1.js", "function_name": "first", "language": "javascript"}
            }
        ]

        chunks2 = [
            {
                "id": "test2",
                "type": "function",
                "content": "function second() { return 'second'; }",
                "metadata": {"file_path": "/test2.js", "function_name": "second", "language": "javascript"}
            }
        ]

        # WHEN
        with IndexerDaemon(temp_path) as daemon:
            result1 = daemon.send_request(chunks1)
            result2 = daemon.send_request(chunks2)

            assert result1["status"] == "success"
            assert result1["indexed_count"] == 1

            assert result2["status"] == "success"
            assert result2["indexed_count"] == 1

        # THEN
        results = search(query="first function", db_path=temp_path)
        assert any("first" in doc for doc in results["documents"][0])

        results = search(query="second function", db_path=temp_path)
        assert any("second" in doc for doc in results["documents"][0])


def describe_index_chunks():
    def test_should_allow_to_index(temp_path, model):
        # GIVEN
        chunks = [
            {
                "id": "auth.js_func_validateToken_15",
                "type": "function",
                "content": "function validateToken(token) { return jwt.verify(token, secret); }",
                "metadata": {
                    "file_path": "/src/auth.js",
                    "function_name": "validateToken",
                    "language": "javascript"
                }
            }
        ]

        # WHEN
        result = index_chunks(req_id=str(uuid.uuid4()), chunks=chunks, model=model, db_path=temp_path)

        # THEN
        assert result["status"] == "success"
        assert result["indexed_count"] == 1

    def test_should_return_the_request_id(temp_path, model):
        # GIVEN
        req_id = str(uuid.uuid4())
        chunks = [
            {
                "id": "auth.js_func_validateToken_15",
                "type": "function",
                "content": "function validateToken(token) { return jwt.verify(token, secret); }",
                "metadata": {
                    "file_path": "/src/auth.js",
                    "function_name": "validateToken",
                    "language": "javascript"
                }
            }
        ]

        # WHEN
        result = index_chunks(req_id=req_id, chunks=chunks, model=model, db_path=temp_path)

        # THEN
        assert result["id"] == req_id

    def test_should_be_able_search_indexed_documents(temp_path, model):
        # GIVEN
        chunks = [
            {
                "id": "auth.js_func_validateToken_15",
                "type": "function",
                "content": "function validateToken(token) { return jwt.verify(token, secret); }",
                "metadata": {
                    "file_path": "/src/auth.js",
                    "function_name": "validateToken",
                    "language": "javascript"
                }
            },
            {
                "id": "utils.py_func_calculateTax_8",
                "type": "function",
                "content": "def calculate_tax(income): return income * 0.3 if income > 50000 else income * 0.2",
                "metadata": {
                    "file_path": "/src/utils.py",
                    "function_name": "calculate_tax",
                    "language": "python"
                }
            }
        ]

        # WHEN
        result = index_chunks(req_id=str(uuid.uuid4()), chunks=chunks, model=model, db_path=temp_path)

        # THEN
        assert result["status"] == "success"

        results = search(query="function to calculate taxes", db_path=temp_path, n_results=1)
        assert len(results["ids"][0]) > 0
        assert "calculate_tax" in results["documents"][0][0]
