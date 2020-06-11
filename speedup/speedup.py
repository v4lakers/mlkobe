# Import Necessary Libraries
import os
import subprocess
import time
from statistics import mean
import matplotlib.pyplot as plt


def main():

    files = ['commands.txt', 'visualizations.txt', 'models.txt']
    N = [1,2,4,6,8]
    dicts = []
    denom = len(N)*len(files)*10
    counter = 0

    # Loop through files
    for file in files:
        temp_dict = {"File Name": file, "data": {}}

        # Loop through Threads
        for n in N:
            sequential_time = []
            parallel_time = []

            # Run 5 Times
            for trial in range(5):
                # Run the Sequential Version
                sequential_start = time.time()
                sequential = subprocess.check_output(["go", "run", "proj3.go", file])
                sequential_time.append(time.time() - sequential_start)
                counter = counter + 1
                print("Percent Complete: {}%".format((counter / denom) * 100), end='\r')

                # Run the Parallel Version
                parallel_start = time.time()
                parallel = subprocess.check_output(["go", "run", "proj3.go", "p="+str(n), file])
                parallel_time.append(time.time() - parallel_start)
                counter = counter + 1
                print("Percent Complete: {}%".format((counter/denom) * 100), end='\r')

            # Get the Average Execution Times
            sequential_average = mean(sequential_time)
            parallel_average = mean(parallel_time)

            # Determine the Speed Up Time
            speedup = sequential_average / parallel_average
            temp_dict["data"][n] = speedup

        # Save the Result
        dicts.append(temp_dict)
        print()

    # Plot the lines
    for dict in dicts:
        data = dict["data"]
        plt.plot(list(data.keys()), list(data.values()), label=dict["File Name"])

    # Plot labels
    plt.xlabel("Number of Threads (N)")
    plt.ylabel("Speedup")
    plt.title("Number of Threads vs. Speedup")
    plt.legend()
    plt.show()

    # Let the User know the Job is done
    print("Complete!")



main()