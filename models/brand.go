package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive" // Import primitive
)

// Brand represents the data structure for a brand in the MongoDB collection
type Brand struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`            // MongoDB primary key
	Name      string             `bson:"name" validate:"required"` // Index this field in MongoDB for lookups
	Details   string             `bson:"details"`
	CreatedAt time.Time          `bson:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt"`
	// Optional: Store filename if you keep the original PDF
	// OriginalPDFPath string `bson:"originalPdfPath,omitempty"`
}

// CreateBrandPayload remains the same as it's for HTTP request binding
type CreateBrandPayload struct {
	Name    string `json:"name" binding:"required"`
	Details string `json:"details" binding:"required"`
}

// UpdateBrandPayload remains the same
type UpdateBrandPayload struct {
	Details string `json:"details" binding:"required"`
}
