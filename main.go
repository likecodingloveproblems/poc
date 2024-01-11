package main

import (
	"encoding/csv"
	"io"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type Products []Product

func getProducts(db *gorm.DB) (Products, error) {
	var products []Product
	if err := db.Find(&products).Error; err != nil {
		return Products{}, err
	}
	return products, nil
}

func render(ctx echo.Context, cmp templ.Component) error {
	return cmp.Render(ctx.Request().Context(), ctx.Response())
}

type parseRow[T any] func(c echo.Context, record []string) (T, error)

func parseProduct(c echo.Context, record []string) (Product, error) {
	price, err := strconv.Atoi(record[1])
	if err != nil {
		return Product{}, err
	}
	product := Product{
		Code:  record[0],
		Price: uint(price),
	}
	return product, nil
}

func GetCsvUploadEndPoint[T any](rowParser parseRow[T], redirectUrl string) echo.HandlerFunc {
	return func(c echo.Context) error {
		file, err := c.FormFile("file")
		if err != nil {
			return err
		}

		// Open the file for reading
		src, err := file.Open()
		if err != nil {
			return err
		}
		defer src.Close()

		// Create a CSV reader
		reader := csv.NewReader(src)

		// Read all records into a slice of Person structs
		var rows []T

		// the first row are columns name
		_, err = reader.Read()
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}

		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return c.String(http.StatusInternalServerError, err.Error())
			}

			// Parse the record into a Person struct
			row, err := rowParser(c, record)
			if err != nil {
				return c.String(http.StatusInternalServerError, err.Error())
			}
			rows = append(rows, row)
		}

		// Create producs in database
		db := getDB()
		db.Create(rows)
		// Send the parsed data as a JSON response
		return c.Redirect(http.StatusFound, redirectUrl)
	}
}

func main() {
	db := getDB()
	e := echo.New()
	e.GET("/admin/products", func(c echo.Context) error {
		products, err := getProducts(db)
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		return render(c, productsAdmin(products))
	})
	e.POST("/admin/products/import", GetCsvUploadEndPoint(parseProduct, "/admin/products"))
	e.Logger.Fatal(e.Start(":8000"))
}
