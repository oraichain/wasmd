# Setup

```bash
# create virtual env
python3 -m venv .venv

# activate venv
source .venv/bin/activate

# install libs
pip install -r indexer/benchmarks/requirements.txt
```

# Load testing

Run load testing using pgbench:

```bash
pgbench -h localhost -p 5432 -U admin -d node_indexer -f ./indexer/dbschema/benchmark.sql -P 1 -c 20 -j 4 -T 20
```

You will receive something like this in the terminal:

```
progress: 1.0 s, 0.0 tps, lat 0.000 ms stddev 0.000, 0 failed
progress: 2.0 s, 4.0 tps, lat 1031.137 ms stddev 110.417, 0 failed
progress: 3.0 s, 18.0 tps, lat 2260.727 ms stddev 369.317, 0 failed
progress: 4.0 s, 5.0 tps, lat 1530.210 ms stddev 752.114, 0 failed
progress: 5.0 s, 3.0 tps, lat 1304.676 ms stddev 102.581, 0 failed
progress: 6.0 s, 20.0 tps, lat 2333.998 ms stddev 538.647, 0 failed
progress: 7.0 s, 4.0 tps, lat 1213.817 ms stddev 215.746, 0 failed
progress: 8.0 s, 14.0 tps, lat 2117.748 ms stddev 415.968, 0 failed
progress: 9.0 s, 9.0 tps, lat 1919.395 ms stddev 790.976, 0 failed
progress: 10.0 s, 4.0 tps, lat 1884.559 ms stddev 619.423, 0 failed
progress: 11.0 s, 13.9 tps, lat 2571.004 ms stddev 445.173, 0 failed
progress: 12.0 s, 6.0 tps, lat 1820.390 ms stddev 712.955, 0 failed
progress: 13.0 s, 13.0 tps, lat 2250.132 ms stddev 446.701, 0 failed
progress: 14.0 s, 8.0 tps, lat 1939.913 ms stddev 527.782, 0 failed
progress: 15.0 s, 7.0 tps, lat 1756.615 ms stddev 669.468, 0 failed
progress: 16.0 s, 14.0 tps, lat 2264.878 ms stddev 454.115, 0 failed
progress: 17.0 s, 6.0 tps, lat 1561.460 ms stddev 732.269, 0 failed
progress: 18.0 s, 15.0 tps, lat 2228.487 ms stddev 486.080, 0 failed
progress: 19.0 s, 6.0 tps, lat 1724.741 ms stddev 672.497, 0 failed
progress: 20.0 s, 12.0 tps, lat 2078.365 ms stddev 630.100, 0 failed
progress: 21.0 s, 11.0 tps, lat 2045.849 ms stddev 689.399, 0 failed
```

Copy those and convert into .csv using any tool you like (like ChatGPT), or call `convert_data.py`

An example of the correct progress_log1.csv format after converting the above output:

```
Time (s),TPS,Latency (ms),Latency Stddev (ms)
1.0,0.0,0.0,0.0
2.0,4.0,1031.137,110.417
3.0,18.0,2260.727,369.317
4.0,5.0,1530.21,752.114
5.0,3.0,1304.676,102.581
6.0,20.0,2333.998,538.647
7.0,4.0,1213.817,215.746
8.0,14.0,2117.748,415.968
9.0,9.0,1919.395,790.976
10.0,4.0,1884.559,619.423
11.0,13.9,2571.004,445.173
12.0,6.0,1820.39,712.955
13.0,13.0,2250.132,446.701
14.0,8.0,1939.913,527.782
15.0,7.0,1756.615,669.468
16.0,14.0,2264.878,454.115
17.0,6.0,1561.46,732.269
18.0,15.0,2228.487,486.08
19.0,6.0,1724.741,672.497
20.0,12.0,2078.365,630.1
21.0,11.0,2045.849,689.399
```

# Start drawing charts

The `draw_chart.py` puts `progress_log1.csv` and `progress_log2.csv` into a chart, so you'll need to generate two .csv files. I did this so that we could have a better comparison between two queries when doing load testing.

After having the two above .csv files, run:

```bash
python indexer/benchmarks/draw_chart.py
```
