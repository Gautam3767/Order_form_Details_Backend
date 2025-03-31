package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Gautam3767/Order_form_Details_Backend.git/database"
	"github.com/Gautam3767/Order_form_Details_Backend.git/models"
	"github.com/Gautam3767/Order_form_Details_Backend.git/services" // Use YOUR module path

	// "github.com/Gautam3767/Order_form_Details_Backend.git/services"
	"github.com/gin-gonic/gin"

	// "github.com/yourusername/brand-service/models"   // Adjust import path
	// "github.com/yourusername/brand-service/services" // Adjust import path

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Context timeout for database operations
const dbTimeout = 5 * time.Second

// ListBrands godoc
// @Summary List all available brand names
// @Description Get a list of all brand names stored in the system
// @Tags brands
// @Produce json
// @Success 200 {array} string "List of brand names"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /brands [get]
func ListBrands(c *gin.Context) {
	coll := database.GetCollection("brands") // Use your collection name env var if needed
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	// Find documents, projecting only the 'name' field, excluding '_id'
	opts := options.Find().SetProjection(bson.M{"name": 1, "_id": 0})
	cursor, err := coll.Find(ctx, bson.M{}, opts) // Empty filter {} means find all

	if err != nil {
		log.Printf("Error finding brands: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve brands"})
		return
	}
	defer cursor.Close(ctx) // Important to close the cursor

	var results []struct { // Temporary struct to decode only the name
		Name string `bson:"name"`
	}
	if err = cursor.All(ctx, &results); err != nil {
		log.Printf("Error decoding brand names: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process brand data"})
		return
	}

	// Extract just the names into a string slice
	brandNames := make([]string, 0, len(results))
	for _, res := range results {
		brandNames = append(brandNames, res.Name)
	}

	// Return empty array instead of null if no brands found
	if brandNames == nil {
		brandNames = []string{}
	}

	c.JSON(http.StatusOK, brandNames)
}

// GetBrandDetails godoc
// @Summary Get details for a specific brand
// @Description Get the stored details associated with a given brand name
// @Tags brands
// @Produce json
// @Param brandName path string true "Name of the brand"
// @Success 200 {object} models.Brand "Brand details"
// @Failure 404 {object} map[string]string "Brand not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /brands/{brandName} [get]
func GetBrandDetails(c *gin.Context) {
	coll := database.GetCollection("brands")
	brandName := c.Param("brandName")
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	var brand models.Brand
	// Find one document where the 'name' field matches
	filter := bson.M{"name": brandName}
	err := coll.FindOne(ctx, filter).Decode(&brand)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Brand '%s' not found", brandName)})
		} else {
			log.Printf("Error finding brand '%s': %v", brandName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error retrieving brand"})
		}
		return
	}

	c.JSON(http.StatusOK, brand)
}

// CreateBrandManual godoc
// @Summary Create a new brand with details (manual entry)
// @Description Add a new brand and its details using a JSON payload
// @Tags brands
// @Accept json
// @Produce json
// @Param brand body models.CreateBrandPayload true "Brand data"
// @Success 201 {object} models.Brand "Brand created successfully"
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 409 {object} map[string]string "Brand already exists (unique name violation)"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /brands [post]
func CreateBrandManual(c *gin.Context) {
	coll := database.GetCollection("brands")
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	var payload models.CreateBrandPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	// Check if brand name already exists (handled by unique index, but good to check first)
	// This check isn't strictly necessary if the index exists and you handle the duplicate key error,
	// but it provides a clearer 409 response before attempting insertion.
	filter := bson.M{"name": payload.Name}
	count, err := coll.CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		log.Printf("Error checking for existing brand '%s': %v", payload.Name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking for existing brand"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Brand '%s' already exists", payload.Name)})
		return
	}

	now := time.Now()
	newBrand := models.Brand{
		// ID will be generated by MongoDB
		Name:      payload.Name,
		Details:   payload.Details,
		CreatedAt: now,
		UpdatedAt: now,
	}

	result, err := coll.InsertOne(ctx, newBrand)
	if err != nil {
		// Handle potential duplicate key error from the unique index
		if mongo.IsDuplicateKeyError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Brand '%s' already exists (database constraint)", payload.Name)})
		} else {
			log.Printf("Error inserting brand '%s': %v", newBrand.Name, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create brand"})
		}
		return
	}

	// Set the ID in the response object
	newBrand.ID = result.InsertedID.(primitive.ObjectID)

	c.JSON(http.StatusCreated, newBrand)
}

// UpdateBrandManual godoc
// @Summary Update details for an existing brand (manual entry)
// @Description Update the details of an existing brand identified by its name
// @Tags brands
// @Accept json
// @Produce json
// @Param brandName path string true "Name of the brand to update"
// @Param details body models.UpdateBrandPayload true "New details data"
// @Success 200 {object} models.Brand "Brand updated successfully"
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 404 {object} map[string]string "Brand not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /brands/{brandName} [put]
func UpdateBrandManual(c *gin.Context) {
	coll := database.GetCollection("brands")
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	brandName := c.Param("brandName")
	var payload models.UpdateBrandPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	filter := bson.M{"name": brandName}
	update := bson.M{
		"$set": bson.M{
			"details":   payload.Details,
			"updatedAt": time.Now(),
		},
	}

	// Option to return the updated document
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var updatedBrand models.Brand
	err := coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updatedBrand)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Brand '%s' not found for update", brandName)})
		} else {
			log.Printf("Error updating brand '%s': %v", brandName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update brand"})
		}
		return
	}

	c.JSON(http.StatusOK, updatedBrand)
}

// UploadBrandPDF godoc
// @Summary Upload a PDF to create or update brand details
// @Description Upload a PDF file. Extracts text and uses it as details. Creates or updates the brand based on 'brandName'.
// @Tags brands
// @Accept multipart/form-data
// @Produce json
// @Param brandName formData string true "Name of the brand"
// @Param pdfFile formData file true "PDF file containing brand details"
// @Success 200 {object} models.Brand "Brand details updated from PDF"
// @Success 201 {object} models.Brand "Brand created from PDF"
// @Failure 400 {object} map[string]string "Bad request (e.g., missing fields, invalid file)"
// @Failure 500 {object} map[string]string "Internal server error (e.g., PDF parsing failed, DB error)"
// @Router /brands/upload [post]
func UploadBrandPDF(c *gin.Context) {
	coll := database.GetCollection("brands")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // Longer timeout for upload+parse+db
	defer cancel()

	// --- 1. Get Form Data (same as before) ---
	brandName := c.PostForm("brandName")
	if brandName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'brandName' form field"})
		return
	}
	fileHeader, err := c.FormFile("pdfFile")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'pdfFile' form field or invalid file upload"})
		return
	}
	// Add validation if desired (file type, size)

	// --- 2. Open and Parse PDF (same as before) ---
	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open uploaded file"})
		return
	}
	defer file.Close()

	extractedText, err := services.ExtractTextFromPDF(file) // Use the chosen parser
	if err != nil {
		log.Printf("Error extracting text from PDF for brand '%s': %v", brandName, err)
		// Handle specific parsing errors as before
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse PDF content."})
		return
	}
	if extractedText == "" {
		log.Printf("Warning: No text extracted from PDF for brand '%s'.", brandName)
		// Decide how to proceed - maybe save empty details or return an informative message
	}

	// --- 3. Upsert Brand in DB ---
	// Upsert = Update if found, Insert if not found
	filter := bson.M{"name": brandName}
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"details":   extractedText,
			"updatedAt": now,
		},
		"$setOnInsert": bson.M{ // Fields to set only when inserting (creating)
			"name":      brandName,
			"createdAt": now,
		},
	}
	opts := options.FindOneAndUpdate().
		SetUpsert(true).                 // Enable Upsert
		SetReturnDocument(options.After) // Return the *new* or *updated* document

	var resultBrand models.Brand
	err = coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&resultBrand)

	if err != nil {
		// Specific upsert errors might need different handling, but generally:
		log.Printf("Error upserting brand '%s' from PDF: %v", brandName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error processing PDF upload"})
		return
	}

	// Determine if it was an insert or update based on timestamps (or check result differently if needed)
	statusCode := http.StatusOK                             // Assume update
	if resultBrand.CreatedAt.Equal(resultBrand.UpdatedAt) { // Approximation: if created == updated, it was likely just inserted
		statusCode = http.StatusCreated
	}

	// --- 4. Return Success Response ---
	c.JSON(statusCode, resultBrand)
}

// DeleteBrand godoc
// @Summary Delete a brand
// @Description Delete a brand by its name
// @Tags brands
// @Produce json
// @Param brandName path string true "Name of the brand to delete"
// @Success 200 {object} map[string]string "Success message"
// @Failure 404 {object} map[string]string "Brand not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /brands/{brandName} [delete]
func DeleteBrand(c *gin.Context) {
	coll := database.GetCollection("brands")
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	brandName := c.Param("brandName")
	filter := bson.M{"name": brandName}

	result, err := coll.DeleteOne(ctx, filter)
	if err != nil {
		log.Printf("Error deleting brand '%s': %v", brandName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete brand"})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Brand '%s' not found", brandName)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Brand '%s' deleted successfully", brandName)})
}
