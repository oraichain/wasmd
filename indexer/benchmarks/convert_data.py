import pandas as pd
import re

# Function to parse the input data from a file
def parse_data_from_file(file_path):
    data = []
    
    with open(file_path, 'r') as file:
        for line in file:
            # Use regex to extract time, tps, latency, and latency stddev
            match = re.match(r"progress: (\d+\.\d+) s, (\d+\.\d+) tps, lat (\d+\.\d+) ms stddev (\d+\.\d+),", line)
            if match:
                time = float(match.group(1))
                tps = float(match.group(2))
                latency = float(match.group(3))
                latency_stddev = float(match.group(4))
                data.append((time, tps, latency, latency_stddev))
    
    return data

# Path to your input file
input_file_path = './indexer/benchmarks/testdata/progress_log1.txt'  # Replace with your file path

# Parse the data from the file
data = parse_data_from_file(input_file_path)

# Create a DataFrame from the data
df = pd.DataFrame(data, columns=["Time (s)", "TPS", "Latency (ms)", "Latency Stddev (ms)"])

# Save the DataFrame to a CSV file
output_csv_path = './indexer/benchmarks/testdata/progress_log1.csv'  # Output CSV file path
df.to_csv(output_csv_path, index=False)

print(f"CSV file '{output_csv_path}' created successfully.")

input_file_path = './indexer/benchmarks/testdata/progress_log2.txt'  # Replace with your file path

# Parse the data from the file
data = parse_data_from_file(input_file_path)

# Create a DataFrame from the data
df = pd.DataFrame(data, columns=["Time (s)", "TPS", "Latency (ms)", "Latency Stddev (ms)"])

# Save the DataFrame to a CSV file
output_csv_path = './indexer/benchmarks/testdata/progress_log2.csv'  # Output CSV file path
df.to_csv(output_csv_path, index=False)

print(f"CSV file '{output_csv_path}' created successfully.")
