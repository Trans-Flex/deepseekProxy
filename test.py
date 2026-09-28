##测试改代理的合并请求是否正常生效
import concurrent.futures
import threading
import time
import requests

TARGET = "http://127.0.0.1:8080/chat"

PAYLOAD = {
    "model": "deepseek-chat",
    "messages": [{"role": "user", "content": "hello"}],
}

N = 50
barrier = threading.Barrier(N)


def one(i):
    barrier.wait()
    start = time.time()
    try:
        r = requests.post(TARGET, json=PAYLOAD, timeout=60)
        return i, r.status_code, time.time() - start
    except Exception as e:
        return i, None, time.time() - start


def main():
    t0 = time.time()
    with concurrent.futures.ThreadPoolExecutor(max_workers=N) as ex:
        results = [f.result() for f in [ex.submit(one, i) for i in range(N)]]
    total = time.time() - t0

    ok = sum(1 for r in results if r[1] == 200)
    times = [r[2] for r in results]

    print(f"并发: {N}  成功: {ok}  总耗时: {total:.3f}s")
    print(f"最快: {min(times):.3f}s  最慢: {max(times):.3f}s  平均: {sum(times)/len(times):.3f}s")


if __name__ == "__main__":
    main()