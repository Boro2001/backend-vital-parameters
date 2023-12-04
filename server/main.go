package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Metric struct {
	Temp      float64 `json:"temp"`
	HR        int     `json:"hr"`
	AccX      float64 `json:"acc_x"`
	AccY      float64 `json:"acc_y"`
	AccZ      float64 `json:"acc_z"`
	Timestamp time.Time
}

const (
	host     = "trumpet.db.elephantsql.com"
	port     = 5432
	user     = "ljamfsdu"
	password = "Xr5PeHu_bHfmWeiL8vFNEglt0t8eGw_E"
	dbname   = "ljamfsdu"
)

var csvMutex sync.Mutex

func writeMetricToCSV(metric Metric) error {
	csvMutex.Lock()
	defer csvMutex.Unlock()

	file, err := os.OpenFile("metrics.csv", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	record := []string{
		fmt.Sprintf("%f", metric.Temp),        // Convert float64 to string
		fmt.Sprintf("%d", metric.HR),          // Convert int to string
		fmt.Sprintf("%f", metric.AccX),        // Convert float64 to string
		fmt.Sprintf("%f", metric.AccY),        // Convert float64 to string
		fmt.Sprintf("%f", metric.AccZ),        // Convert float64 to string
		metric.Timestamp.Format(time.RFC3339), // Use RFC3339 format for the timestamp
	}

	err = writer.Write(record)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	connStr := fmt.Sprintf("postgres://%s:%s@%s/%s", user, password, host, dbname)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	router := gin.Default()

	router.POST("/metrics", func(c *gin.Context) {
		var metric Metric
		if err := c.ShouldBindJSON(&metric); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		metric.Timestamp = time.Now()

		_, err := db.Exec("INSERT INTO metrics (temp, hr, acc_x, acc_y, acc_z, timestamp) VALUES ($1, $2, $3, $4, $5, $6)",
			metric.Temp, metric.HR, metric.AccX, metric.AccY, metric.AccZ, metric.Timestamp)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, metric)
	})

	router.GET("/metrics", func(c *gin.Context) {
		timestampFrom := c.Query("timestamp_from")
		timestampTo := c.Query("timestamp_to")

		query := "SELECT temp, hr, acc_x, acc_y, acc_z, timestamp FROM metrics"
		whereClauses := []string{}

		if timestampFrom != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("timestamp >= '%s'", timestampFrom))
		}
		if timestampTo != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("timestamp <= '%s'", timestampTo))
		}

		if len(whereClauses) > 0 {
			query += " WHERE " + strings.Join(whereClauses, " AND ")
		}

		// Add an ORDER BY and LIMIT clause to get the latest 100 records if no timestamp range is provided
		if timestampFrom == "" && timestampTo == "" {
			query += " ORDER BY timestamp DESC LIMIT 100"
		}

		rows, err := db.Query(query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var metrics []Metric
		for rows.Next() {
			var m Metric
			err := rows.Scan(&m.Temp, &m.HR, &m.AccX, &m.AccY, &m.AccZ, &m.Timestamp)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			metrics = append(metrics, m)
		}

		if err = rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, metrics)
	})

	router.POST("/metrics_csv", func(c *gin.Context) {
		var metric Metric
		if err := c.ShouldBindJSON(&metric); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		metric.Timestamp = time.Now()

		// Write the metric to a CSV file
		if err := writeMetricToCSV(metric); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, metric)
	})

	router.GET("/metrics_csv", func(c *gin.Context) {
		file, err := os.Open("metrics.csv")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to read CSV file"})
			return
		}
		defer file.Close()

		csvReader := csv.NewReader(file)
		records, err := csvReader.ReadAll()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reading CSV records"})
			return
		}

		var metrics []Metric
		for _, record := range records {
			temp, _ := strconv.ParseFloat(record[0], 64)
			hr, _ := strconv.Atoi(record[1])
			accX, _ := strconv.ParseFloat(record[2], 64)
			accY, _ := strconv.ParseFloat(record[3], 64)
			accZ, _ := strconv.ParseFloat(record[4], 64)
			timestamp, _ := time.Parse(time.RFC3339, record[5])

			metric := Metric{
				Temp:      temp,
				HR:        hr,
				AccX:      accX,
				AccY:      accY,
				AccZ:      accZ,
				Timestamp: timestamp,
			}
			metrics = append(metrics, metric)
		}

		c.JSON(http.StatusOK, metrics)
	})
	router.Run(":8080")
}
