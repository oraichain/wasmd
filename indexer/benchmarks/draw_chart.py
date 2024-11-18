import pandas as pd
import matplotlib.pyplot as plt

# Load the two CSV files
df1 = pd.read_csv('./indexer/benchmarks/testdata/progress_log1.csv')  # Replace with the path to your first CSV file
df2 = pd.read_csv('./indexer/benchmarks/testdata/progress_log2.csv')  # Replace with the path to your second CSV file

# Plotting TPS and Latency from both CSVs
plt.figure(figsize=(10, 6))

# Plotting TPS for both datasets
plt.subplot(2, 1, 1)
plt.plot(df1['Time (s)'], df1['TPS'], label='TPS (many non-height filter conditions)', color='blue', marker='o')
plt.plot(df2['Time (s)'], df2['TPS'], label='TPS (only one non-height filter condition)', color='green', marker='x')
plt.xlabel('Time (s)')
plt.ylabel('TPS')
plt.title('TPS Comparison: File 1 vs File 2')
plt.legend()
plt.grid(True)

# Plotting Latency for both datasets
plt.subplot(2, 1, 2)
plt.plot(df1['Time (s)'], df1['Latency (ms)'], label='Latency (many non-height filter conditions)', color='red', marker='o')
plt.plot(df2['Time (s)'], df2['Latency (ms)'], label='Latency (only one non-height filter condition)', color='orange', marker='x')
plt.xlabel('Time (s)')
plt.ylabel('Latency (ms)')
plt.title('Latency Comparison: File 1 vs File 2')
plt.legend()
plt.grid(True)

# Show the plots
plt.tight_layout()
plt.show()
