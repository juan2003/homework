package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strconv"
)

//Field names
const (
	Plan_ID         = 0
	Plan_State      = 1
	Plan_MetalLevel = 2
	Plan_Rate       = 3
	Plan_RateArea   = 4
	Zip_Code        = 0
	Zip_State       = 1
	Zip_Fips        = 2
	Zip_County      = 3
	Zip_RateArea    = 4
	Slcsp_Zip       = 0
	Slcsp_Rate      = 1
)

type IndexFunction func(i, j *[]string) bool

type TableIndex struct {
	Index           []*[]string
	CompareFunction IndexFunction
}

func (ti *TableIndex) Len() int {
	return len(ti.Index)
}
func (ti *TableIndex) Swap(i, j int) {
	ti.Index[i], ti.Index[j] = ti.Index[j], ti.Index[i]
}
func (ti *TableIndex) Less(i, j int) bool {
	return ti.CompareFunction(ti.Index[i], ti.Index[j])
}

type BaseTable struct {
	Header    []string
	Records   [][]string
	Index     TableIndex
	IsIndexed bool
}

func (bt *BaseTable) Load(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		defer file.Close()
		reader := csv.NewReader(file)
		bt.Header, err = reader.Read()
		if err == nil {
			bt.Records, err = reader.ReadAll()
			bt.IsIndexed = false
		}
	}
	return err
}

func (bt *BaseTable) Save(filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		defer file.Close()
		writer := csv.NewWriter(file)
		err = writer.Write(bt.Header)
		if err == nil {
			err = writer.WriteAll(bt.Records)
		}
	}
	return err
}

func (bt *BaseTable) BuildIndex(cf IndexFunction) {
	//Set the compare function
	bt.Index.CompareFunction = cf
	//Build the index
	bt.Index.Index = make([]*[]string, len(bt.Records))
	for i := range bt.Records {
		bt.Index.Index[i] = &bt.Records[i]
	}
	sort.Sort(&bt.Index)
	bt.IsIndexed = true
}

func (bt *BaseTable) ListSorted() {
	if !bt.IsIndexed {
		fmt.Println("Table is not indexed")
		return
	}
	for _, i := range bt.Index.Index {
		fmt.Println(*i)
	}
}

func GetRateArea(zipcode string, zipTable BaseTable) (string, string) {
	rowcount := len(zipTable.Records)
	zipIndex := sort.Search(rowcount, func(i int) bool { return zipcode <= (*zipTable.Index.Index[i])[Zip_Code] })
	if zipIndex < rowcount {
		zipRow := *zipTable.Index.Index[zipIndex]
		state, rateArea := zipRow[Zip_State], zipRow[Zip_RateArea]

		//Before we return, confirm that other entries hold same data
		for (*zipTable.Index.Index[zipIndex])[Zip_Code] == zipcode && zipIndex < rowcount {
			if state != (*zipTable.Index.Index[zipIndex])[Zip_State] || rateArea != (*zipTable.Index.Index[zipIndex])[Zip_RateArea] {
				return "", ""
			}
			zipIndex++
		}
		return state, rateArea
	}
	return "", ""
}

func GetSecondLowestCostPlan(state string, rateArea string, planTable BaseTable) string {
	// metal level is defined to be silver
	const metalLevel = "Silver"
	rowcount := len(planTable.Records)
	planIndex := sort.Search(rowcount, func(i int) bool {
		currentRow := *planTable.Index.Index[i]
		switch {
		case currentRow[Plan_State] != state:
			return state < currentRow[Plan_State]
		case currentRow[Plan_RateArea] != rateArea:
			return rateArea < currentRow[Plan_RateArea]
		default:
			return metalLevel <= currentRow[Plan_MetalLevel]
		}
	})
	if planIndex < rowcount {
		//the first item is the lowest cost plan
		lowestCostPlan := *planTable.Index.Index[planIndex]

		//Increment to the next plan with a different rate
		for (*planTable.Index.Index[planIndex])[Plan_Rate] == lowestCostPlan[Plan_Rate] && planIndex < rowcount {
			planIndex++
		}

		if planIndex < rowcount {
			//Confirm that the state, rate area, and metal level are the same
			secondLowestCostPlan := *planTable.Index.Index[planIndex]
			if secondLowestCostPlan[Plan_State] == state && secondLowestCostPlan[Plan_RateArea] == rateArea && secondLowestCostPlan[Plan_MetalLevel] == metalLevel {
				return secondLowestCostPlan[Plan_Rate]
			}
		}
	}
	return ""
}

func main() {
	// Load the data from csv
	var plans, zips, slcsp BaseTable
	plans.Load("plans.csv")
	zips.Load("zips.csv")
	slcsp.Load("slcsp.csv")
	// Create the comparer function
	// "index" (sort) the table by state, rate_area, metal_level, and rate
	plans.BuildIndex(func(i, j *[]string) bool {
		switch {
		case (*i)[Plan_State] != (*j)[Plan_State]:
			return (*i)[Plan_State] < (*j)[Plan_State]
		case (*i)[Plan_RateArea] != (*j)[Plan_RateArea]:
			//Note that RateArea will be string sorted instead of numeric sorted
			//In this case, it does not matter, as we only care that the areas are separated
			return (*i)[Plan_RateArea] < (*j)[Plan_RateArea]
		case (*i)[Plan_MetalLevel] != (*j)[Plan_MetalLevel]:
			return (*i)[Plan_MetalLevel] < (*j)[Plan_MetalLevel]
		default:
			//Convert rate to float before comparing
			rateI, _ := strconv.ParseFloat((*i)[Plan_Rate], 32)
			rateJ, _ := strconv.ParseFloat((*j)[Plan_Rate], 32)
			return rateI < rateJ
		}
	})
	//Index by zip code
	zips.BuildIndex(func(i, j *[]string) bool {
		return (*i)[Zip_Code] < (*j)[Zip_Code]
	})

	//create cache
	cache := make(map[string]string, 100)

	for n := range slcsp.Records {
		zip := slcsp.Records[n][Slcsp_Zip]
		// looking up a zip code will provide state and rate_area
		state, rateArea := GetRateArea(zip, zips)
		if state != "" && rateArea != "" {
			cacheKey := state + "_" + rateArea
			rate, cacheHit := cache[cacheKey]
			if !cacheHit {
				rate = GetSecondLowestCostPlan(state, rateArea, plans)
				cache[cacheKey] = rate
			}
			slcsp.Records[n][Slcsp_Rate] = rate
		}
	}

	plans.ListSorted()
	slcsp.Save("output.csv")

}
