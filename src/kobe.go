package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/asaskevich/govalidator"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg/draw"
	"image/color"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

func printUsage() {
	commands := "Parallel Usage: go run proj3.go p=[number of goroutines] <[Name of textfile]\n" +
		"Sequential Usage: go run proj3.go <[Name of textfile]\n"

	fmt.Println(commands)
}

// Struct for a Visualization Job from txt file in JSON format
type VisualizeJob struct {
	Command string `json:"command"`
	Specs []string `json:"specs"`
}

// Struct for an ML Job from txt file in JSON format
type MLJob struct {
	Command string `json:"command"`
	Model string `json:"model"`
	Parameters []string `json:"params"`
	ParameterRange [][]interface{} `json:"param_range"`
}

// Struct for a single Machine Learning Model
type MLRequest struct {

	Model string
	Parameters []string
	ParameterValues string
	Accuracy float64
}


// Struct for all the attributes of a shot by Kobe Bryant
type Shot struct {
	ActionType string `json:"action_type"`
	CombinedShotType string `json:"combined_shot_type"`
	GameEventID int `json:"game_event_id"`
	GameId int `json:"game_id"`
	Lat float64 `json:"lat"`
	X float64 `json:"loc_x"`
	Y float64 `json:"loc_y"`
	Lon float64 `json:"lon"`
	MinutesRemaining int `json:"minutes_remaining"`
	Period int `json:"period"`
	Playoffs int `json:"playoffs"`
	Season string `json:"season"`
	SecondsRemaining int `json:"seconds_remaining"`
	ShotDistance int `json:"shot_distance"`
	ShotMadeFlag int `json:"shot_made_flag"`
	ShotType string `json:"shot_type"`
	ShotZoneArea string `json:"shot_zone_area"`
	ShotZoneBasic string `json:"shot_zone_basic"`
	ShotZoneRange string `json:"shot_zone_range"`
	TeamId int `json:"team_id"`
	TeamName string `json:"team_name"`
	GameDate string `json:"game_date"`
	Location string `json:"location"`
	Opponent string `json:"opponent"`
	ShotID int `json:"shot_id"`

}

/*
	Implementing a Lock Free Queue
	Following Code from Lamont Samuels
	(Specifically from Problem 6 of the Midterm)
	And Adapting to include Strings
	Used: https://go101.org/article/unsafe.html
	for all things related to the unsafe pointer
 */


// Struct for the Queue
type Queue struct {
	values []*unsafe.Pointer
	end  int32
}

// Function that gives a ticket number and increments
func (queue *Queue) getAndIncrement() int32 {
	ticket := atomic.AddInt32(&queue.end, 1)
	return ticket - 1
}

// Function that switches pointer for an item that is to be dequeue'd
func (queue *Queue) getAndSet(curr *unsafe.Pointer, new unsafe.Pointer) unsafe.Pointer {
	prev := atomic.SwapPointer(curr, new)
	return prev
}

// Function that gets a ticket and assigns a position to the enqueue value
func (queue *Queue) enq(value *unsafe.Pointer) {
	i := queue.getAndIncrement()
	queue.values[i] = value
}

// Function that Grabs the first valid value from the Queue
func (queue *Queue) deq() string {

	// While True
	for {
		size := queue.end

		// Loop over the elements of the Queue
		for i := int32(0); i < size; i++ {

			// New value our pointer will point to
			invalid := "fake"

			// Set the pointer to a fake value
			tempValue := queue.getAndSet(queue.values[i], unsafe.Pointer(&invalid))

			// Get the Value the pointer previously pointed to
			value := *(*string)(tempValue)

			// Return if this was a valid value
			if value != "fake" {
				return value
			}
		}
	}
}

// Function that returns a Queue
func NewQueue() Queue{
	return Queue{values: make([]*unsafe.Pointer,50), end: 0,}
}

// Function that carries out the visualization job
func visualization(lo int, hi int,p *plot.Plot, shots []Shot, Specs []string, m map[string]int, plottingComplete chan bool, parallel bool){

	// XY's for Shots made
	var xysMake plotter.XYs

	// XY's for Shots missed
	var xysMiss plotter.XYs

	// Loop through specified shots
	for i:=lo; i<hi; i++{

		// Boolean that determines if we should plot or not
		write := true

		// Followed Advice Found on StackOverFlow:
		// https://stackoverflow.com/questions/20170275/how-to-find-the-type-of-an-object-in-go
		// Get the Value
		f := reflect.ValueOf(&shots[i]).Elem()

		// Loop through the Specification the User Specifies
		for j:=0; j<len(Specs); j++{

			// Get the Key and Value
			temp := strings.Split(Specs[j], ":")

			// Use our Mapping to get the Key
			key := m[strings.TrimSpace(temp[0])]

			// Followed Advice Found on StackOverFlow:
			// https://stackoverflow.com/questions/54191322/go-reflect-field-name-to-specific-interface
			dataValue := f.Field(key).Interface()

			// Strip the white spaces
			temp_val := strings.TrimSpace(temp[1])

			// Using Function Made by Asaskvich
			// Github Page: https://github.com/asaskevich/govalidator
			// If our Value is an Integer
			if govalidator.IsInt(temp_val){
				value, _ := strconv.Atoi(temp[1])

				// If the data point doesn't meet the criteria
				// Set write bool to false, exit for loop
				if value != dataValue {
					write = false
					break
				}

			// If our Value is a string
			}else{
				value := strings.TrimSpace(temp[1])

				// If the data point doesn't meet the criteria
				// Set write bool to false, exit for loop
				if value != dataValue {
					write = false
					break
				}
			}

		}
		// If our data point meets all the criteria, plot
		if write{

			// If the Shot was Made, add to xysMake array
			if shots[i].ShotMadeFlag == 1{
				x := shots[i].X
				y := shots[i].Y
				xysMake = append(xysMake, struct{X, Y float64 }{x,y})

			// If the Shot was Missed, add to the xysMiss array
			}else{
				x := shots[i].X
				y := shots[i].Y
				xysMiss = append(xysMiss, struct{X, Y float64 }{x,y})
			}

		}


	}

	// Plot our Resulting Data Points
	// Code Inspired By justforfunc: Programming in Go (Campoy)
	// Youtube Video: https://www.youtube.com/watch?v=ihP7lQivA6M
	// Github Page: https://github.com/campoy/justforfunc/tree/master/34-gonum-plot

	// Turn our Miss array into XY points to plot
	miss, _ := plotter.NewScatter(plotter.XYs(xysMiss))

	// Format the Points to be purple X's
	miss.GlyphStyle = draw.GlyphStyle{
		Color:  color.RGBA{R:137, G:68, B:202, A:255},
		Radius: 5.5,
		Shape:  draw.CrossGlyph{},

	}
	// Add the Points to the Plot
	p.Add(miss)

	// Turn our array into XY points to plot
	made, _ := plotter.NewScatter(plotter.XYs(xysMake))

	// Format the Point to be Gold O's
	made.GlyphStyle = draw.GlyphStyle{
		Color:  color.RGBA{R:253, G:185, B:39, A:255},
		Radius: 4.5,
		Shape:  draw.RingGlyph{},
	}
	// Add the Points to the Plot
	p.Add(made)


	// If this was done in parallel,
	// Synchronize with the Channel
	if parallel{
		plottingComplete <- true
	}

}

// Worker Function that handles the pre/post processing of a visualization job
func visualizationWorker(str string, threads int, wg *sync.WaitGroup, parallel bool, shots []Shot, m map[string]int){

	// Create a new Plot Object p
	p, err := plot.New()
	if err != nil{
		log.Fatal(err)
	}

	// Turn the Job String to JSON Object
	var job VisualizeJob
	_ = json.Unmarshal([]byte(str), &job)

	// Variable that will be used to name outfile
	var outfile string

	// Channel to synchronize threads
	plottingComplete := make(chan bool, threads)

	// If parallel Version
	if parallel{

		// All of Kobe's Shots
		records := len(shots)

		// Partitions based on threads
		partitions := records/threads

		// First Partition Range
		lo := 0
		hi := partitions

		// Spawn a go routine for each thread
		for i:=0; i<threads; i++{

			// Spawn Go Routine
			go visualization(lo, hi, p, shots, job.Specs, m, plottingComplete, parallel)


			// Update the next partition's range
			lo += partitions

			// Last thread takes all remaining tasks
			if i == threads-2{
				hi = len(shots)

			}else{
				hi += partitions
			}

		}


	// Sequential Version
	} else{

		// Call visualization function for all the shots
		visualization(0, len(shots),p, shots, job.Specs, m, plottingComplete, parallel)

	}

	// Loop through the Specs to get
	// an appropriate outfile name
	for j:=0; j<len(job.Specs); j++{
		if j != len(job.Specs)-1{
			outfile = outfile+job.Specs[j]+" "

		} else{
			outfile = outfile+job.Specs[j]
		}

	}

	// In the case we want all shots to be displayed
	if len(outfile) == 0{
		outfile = "All_Shots"
	}

	// Add a Legend
	p.Legend.Add("X = Miss")
	p.Legend.Add("O = Make")
	p.Legend.Color = color.RGBA{R:255, G:255, B:255, A:255}
	p.Legend.Top = true
	p.Legend.Font.Size = 16

	// Background Color
	p.BackgroundColor = color.RGBA{R:0, G:0, B:0, A:255}

	// Title
	p.Title.Text = outfile
	p.Title.Color = color.RGBA{R:255, G:255, B:255, A:255}
	p.Title.Font.Size = 20


	// Code Inspired By justforfunc: Programming in Go (Campoy)
	// Youtube Video: https://www.youtube.com/watch?v=ihP7lQivA6M
	// Github Page: https://github.com/campoy/justforfunc/tree/master/34-gonum-plot

	// Create a file in our visualization Results Directory
	outfile = outfile+".png"
	dir := "visualizationResults"
	f, _ := os.Create(filepath.Join(dir, filepath.Base(outfile)))

	// If parallel, wait for all threads to finish their jobs
	if parallel{
		for t:=1; t<=threads; t++{
			<- plottingComplete
		}

	}

	// Code Below By justforfunc: Programming in Go (Campoy)
	// Youtube Video: https://www.youtube.com/watch?v=ihP7lQivA6M
	// Github Page: https://github.com/campoy/justforfunc/tree/master/34-gonum-plot

	// Write our plot to our file and save
	wt, _ := p.WriterTo(700, 500, "png")
	_, _ = wt.WriteTo(f)

	// If Parallel, signal that we are done
	if parallel{
		wg.Done()

	}

}

// Worker function that handles the pre/post processing of a ML Job
func mlWorker(str string, threads int, wg *sync.WaitGroup, parallel bool){

	// Convert the string to JSON format
	var job MLJob
	json.Unmarshal([]byte(str), &job)


	// Loop through the parameters to check which are numeric
	// We will be turning int slices like [1,4] -> [1,2,3,4]
	for i:=0; i<len(job.ParameterRange); i++{

		// If the data is numeric
		if "float64" == reflect.TypeOf(job.ParameterRange[i][0]).String(){

			// Get the Min of the range
			min := job.ParameterRange[i][0].(float64)
			// Get the Max of the range
			top := job.ParameterRange[i][1].(float64)

			// Loop through all the values between min and max
			// and append to the array
			for j := min; j<=top; j++{
				job.ParameterRange[i] = append(job.ParameterRange[i], j)

			}

			// Get rid of the first two values (our original min/max values)
			job.ParameterRange[i] = job.ParameterRange[i][2:]
		}

	}

	// Create an array that holds ML Request Objects
	// Each object will correspond to a specific model with specific parameters
	var requests []MLRequest

	// Loop through all the parameters
	// The if/else blocks will dictate where to go
	// depending on how many parameters the user gives
	for i:=0; i<len(job.ParameterRange[0]); i++{

		// If the user specifies more than 1 parameter
		if len(job.ParameterRange) > 1{

			// If the user Specifies 2 or more parameters
			for j:=0; j<len(job.ParameterRange[1]); j++{
				if len(job.ParameterRange) > 2{

					// If the user Specifies 3 parameters
					for k:=0; k<len(job.ParameterRange[2]); k++{
						request := MLRequest{
							Model:           job.Model,
							Parameters:      job.Parameters,
							ParameterValues: fmt.Sprint(job.ParameterRange[0][i])+"_"+fmt.Sprint(job.ParameterRange[1][j])+"_"+fmt.Sprint(job.ParameterRange[2][k]),
						}

						requests = append(requests, request)

					}

				// If the user Specifies 2 parameters
				}else{
					request := MLRequest{
						Model:           job.Model,
						Parameters:      job.Parameters,
						ParameterValues: fmt.Sprint(job.ParameterRange[0][i])+"_"+fmt.Sprint(job.ParameterRange[1][j]),
					}

					requests = append(requests, request)
				}

			}

		// If the user Specifies 1 parameter
		}else{
			request := MLRequest{
				Model:           job.Model,
				Parameters:      job.Parameters,
				ParameterValues: fmt.Sprint(job.ParameterRange[0][i]),
			}

			requests = append(requests, request)
		}

	}

	// Variable to hold best model
	var globalMax float64
	var bestModel MLRequest

	// If we are running in parallel
	if parallel{
		var size int

		// Channel that will be used for synchronization
		threadComplete := make(chan MLRequest, threads)

		// If there are more models to test than threads
		if len(requests) > threads{

			// Get the Partition
			y := len(requests)
			partitions := y/threads

			// Range of first partition
			lo := 0
			hi:=partitions

			// Spawn a go routine for each thread
			for i:=0; i<threads; i++{
				go mlParallel(lo, hi, requests, threadComplete)

				// Update the range of our partitions
				lo += partitions

				// For the last thread, take all remaining tasks
				if i == threads-2{
					hi = len(requests)

				}else{
					hi += partitions
				}
			}
			size = threads

		// If there are more thread than requests
		}else{
			// Spawn go routines according the amount of request
			for i:=0;i<len(requests);i++{
				go mlParallel(i, i+1, requests, threadComplete)
			}
			size = len(requests)

		}

		// Synchronize go routines to wait here before moving on
		// This chunk also determines the best model to use
		for j:=0; j<size; j++{

			// Get the model accuracy
			temp := <- threadComplete

			// Update max accordingly
			if temp.Accuracy > globalMax{
				globalMax = temp.Accuracy
				bestModel = temp
			}
		}

	// Sequential Version
	} else{
		// Loop through requests
		for i:=0; i<len(requests); i++{

			// Call the Python Function
			// Learned how to do this from:
			// https://stackoverflow.com/questions/27021517/go-run-external-python-script
			cmd,_ := exec.Command("python", "src/ml.py", requests[i].Model, fmt.Sprint(requests[i].Parameters), requests[i].ParameterValues).Output()

			// Get the results and process them
			result := strings.TrimSuffix(string(cmd), "\n")
			temp,_ := strconv.ParseFloat(result, 64)

			// Update max accordingly
			if temp > globalMax{
				globalMax = temp
				bestModel = requests[i]
			}

		}

	}

	// Print the Best Model, along with parameters and accuracy,  to the user
	bestModel.Accuracy = globalMax
	fmt.Println("Model: ",bestModel.Model)
	best_params := strings.Split(bestModel.ParameterValues,"_")
	for i:=0;i<len(bestModel.Parameters);i++{
		fmt.Println(bestModel.Parameters[i]+" : "+best_params[i])
	}
	fmt.Println("Accuracy: ",bestModel.Accuracy)
	fmt.Println()

	// If running parallel, signal done
	if parallel{
		wg.Done()
	}

}

// Function that carries out ml job for parallel version
func mlParallel(lo int, hi int, requests []MLRequest, threadComplete chan MLRequest){

	// Local Max for a thread
	var max float64
	var bestModel MLRequest

	// Loop through the requests in the range given
	for i:=lo; i<hi; i++{

		// Call the python function
		// Learned how to do this from:
		// https://stackoverflow.com/questions/27021517/go-run-external-python-script
		cmd,_ := exec.Command("python", "src/ml.py", requests[i].Model, fmt.Sprint(requests[i].Parameters), requests[i].ParameterValues).Output()

		// Get and process the result
		result := strings.TrimSuffix(string(cmd), "\n")
		temp,_ := strconv.ParseFloat(result, 64)

		// Update the max accordingly
		if temp > max{
			max = temp
			bestModel = requests[i]
		}

	}

	// Let the main thread know this thread is done
	// Also send the best model found
	bestModel.Accuracy = max
	threadComplete <- bestModel

}

// Reader function that process the standard input
func reader(threads int, parallel bool, queue Queue, wg *sync.WaitGroup, shots []Shot, m map[string]int) {

	// Add jobs from the input to the queue
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan(){
		tempJob := scanner.Text()
		tempJobP := unsafe.Pointer(&tempJob)
		queue.enq(&tempJobP)

	}

	// Boolean that signals when to stop reading from queue
	var done bool

	// Variable that checks if we have gone through all items from the queue
	counter := int32(0)

	// Value to reach before signaling done to be true
	tail := queue.end

	// Dequeue until done is true
	for !done{

		// If there are still tasks to read
		if counter != tail{

			// Grab a job
			temp := queue.deq()

			// Visualization Job
			if strings.Contains(temp, "visualize"){

				// Run Parallel Version
				if parallel{
					wg.Add(1)
					go visualizationWorker(temp, threads, wg, parallel, shots, m)

					// Run Sequential Version
				} else{
					visualizationWorker(temp, threads, wg, parallel, shots, m)
				}
			// ML Model Job
			} else if strings.Contains(temp, "ml"){
				
				// Run Parallel Version
				if parallel{
					wg.Add(1)
					go mlWorker(temp, threads, wg, parallel)

				// Run Sequential Version
				}else{
					mlWorker(temp, threads, wg, parallel)

				}

			}
			// Increment Counter
			counter++
		
		// We have read everything
		// break out of for loop
		} else {

			done = true
		}

	}

}

// Main function that runs this bad boy
func main() {

	// Threads
	var threads int

	// If running parallel or sequential
	var parallel bool

	// Create a queue to hold the jobs
	queue := NewQueue()

	// Make a directory to hold results of visualization jobs
	os.Mkdir("visualizationResults", 0775)

	// Open the JSON File and save contents to variable shots
	// this will hold all the shots kove bryant has ever took
	data, _ := ioutil.ReadFile("data/data.json")
	shots := make([]Shot,0)
	_ = json.Unmarshal(data, &shots)

	// Create a mapping of the Shot Struct
	// This helps us in accessing the data in future steps
	// https://blog.golang.org/maps
	var temp Shot
	e := reflect.ValueOf(&temp).Elem()
	m := make(map[string]int)

	// Assign a number to each field for the struct
	for i := 0; i < e.NumField(); i++ {
		varName := e.Type().Field(i).Name
		m[varName] = i
	}

	// If the User provides parallel flag
	input := os.Args
	if len(input) == 2{

		// Get the amount of threads
		temp1 := os.Args[1]
		temp2 := strings.Split(temp1, "=")
		i, err := strconv.Atoi(temp2[1])
		threads = i
		if err != nil {
			fmt.Println(err)
		}

		// Set GOMAXPROCS
		runtime.GOMAXPROCS(threads)
		parallel = true
	}

	// Print Usage for bad input
	if len(input) != 1 && len(input) != 2{
		printUsage()
		os.Exit(3)

	}


	// Specify the amount of readers
	readers := int(math.Ceil(float64(threads)*(1.0/5.0)))

	// Create a wait group
	wg := sync.WaitGroup{}

	// If running in parallel
	if parallel{

		// Spawn appropriate amount of readers
		for i:=0;i<readers;i++{
			go reader(threads, parallel, queue, &wg, shots, m)

		}
		// Wait for a couple seconds so the other threads
		// have a chance to begin working
		time.Sleep(time.Second*2)

	// If running Sequentially
	}else{
		reader(threads, parallel, queue, &wg, shots, m)
	}

	// Wait to end function until all jobs are done
	wg.Wait()
}