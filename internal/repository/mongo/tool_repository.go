package mongo

import (
	"context"
	"errors"
	"fmt"

	domain "tool_management_backend/internal/domain/tool"
	"tool_management_backend/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ToolRepository struct {
	collection *mongo.Collection
}

type toolDocument struct {
	ID                primitive.ObjectID          `bson:"_id,omitempty"`
	Code              string                      `bson:"code"`
	Name              string                      `bson:"name"`
	Type              string                      `bson:"type"`
	Status            string                      `bson:"status"`
	Responsible       string                      `bson:"responsible"`
	AssignmentDate    string                      `bson:"assignmentDate"`
	DateMaintenance   string                      `bson:"dateMaintenance"`
	NextMaintenance   string                      `bson:"nextMaintenance"`
	Location          string                      `bson:"location"`
	Notes             string                      `bson:"notes"`
	Deterioration     bool                        `bson:"deterioration"`
	AssignmentHistory []domain.AssignmentHistory  `bson:"assignmentHistory"`
	MaintenanceRecord []domain.MaintenanceHistory `bson:"maintenanceRecord"`
}

func NewToolRepository(collection *mongo.Collection) *ToolRepository {
	return &ToolRepository{collection: collection}
}

func (r *ToolRepository) Create(ctx context.Context, tool domain.Tool) (domain.Tool, error) {
	document := toDocument(tool)
	document.ID = primitive.NewObjectID()

	if _, err := r.collection.InsertOne(ctx, document); err != nil {
		return domain.Tool{}, fmt.Errorf("create tool: %w", err)
	}

	return toDomain(document), nil
}

func (r *ToolRepository) List(ctx context.Context, filter domain.Filter) ([]domain.Tool, error) {
	cursor, err := r.collection.Find(ctx, buildFilter(filter))
	if err != nil {
		return nil, fmt.Errorf("list tools: %w", err)
	}
	defer cursor.Close(ctx)

	var documents []toolDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, fmt.Errorf("decode tools: %w", err)
	}

	tools := make([]domain.Tool, 0, len(documents))
	for _, document := range documents {
		tools = append(tools, toDomain(document))
	}

	return tools, nil
}

func (r *ToolRepository) GetByID(ctx context.Context, id string) (domain.Tool, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return domain.Tool{}, fmt.Errorf("invalid tool id: %w", err)
	}

	var document toolDocument
	if err := r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&document); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.Tool{}, repository.ErrNotFound
		}
		return domain.Tool{}, fmt.Errorf("get tool: %w", err)
	}

	return toDomain(document), nil
}

func (r *ToolRepository) Update(ctx context.Context, tool domain.Tool) (domain.Tool, error) {
	objectID, err := primitive.ObjectIDFromHex(tool.ID)
	if err != nil {
		return domain.Tool{}, fmt.Errorf("invalid tool id: %w", err)
	}

	document := toDocument(tool)
	document.ID = objectID

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, bson.M{"$set": document})
	if err != nil {
		return domain.Tool{}, fmt.Errorf("update tool: %w", err)
	}
	if result.MatchedCount == 0 {
		return domain.Tool{}, repository.ErrNotFound
	}

	return tool, nil
}

func (r *ToolRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid tool id: %w", err)
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return fmt.Errorf("delete tool: %w", err)
	}
	if result.DeletedCount == 0 {
		return repository.ErrNotFound
	}

	return nil
}

func toDocument(tool domain.Tool) toolDocument {
	return toolDocument{
		Code:              tool.Code,
		Name:              tool.Name,
		Type:              tool.Type,
		Status:            tool.Status,
		Responsible:       tool.Responsible,
		AssignmentDate:    tool.AssignmentDate,
		DateMaintenance:   tool.DateMaintenance,
		NextMaintenance:   tool.NextMaintenance,
		Location:          tool.Location,
		Notes:             tool.Notes,
		Deterioration:     tool.Deterioration,
		AssignmentHistory: tool.AssignmentHistory,
		MaintenanceRecord: tool.MaintenanceRecord,
	}
}

func toDomain(document toolDocument) domain.Tool {
	return domain.Tool{
		ID:                document.ID.Hex(),
		Code:              document.Code,
		Name:              document.Name,
		Type:              document.Type,
		Status:            document.Status,
		Responsible:       document.Responsible,
		AssignmentDate:    document.AssignmentDate,
		DateMaintenance:   document.DateMaintenance,
		NextMaintenance:   document.NextMaintenance,
		Location:          document.Location,
		Notes:             document.Notes,
		Deterioration:     document.Deterioration,
		AssignmentHistory: document.AssignmentHistory,
		MaintenanceRecord: document.MaintenanceRecord,
	}
}

func buildFilter(filter domain.Filter) bson.M {
	query := bson.M{}

	if filter.Status != "" {
		query["status"] = filter.Status
	}
	if filter.Responsible != "" {
		query["responsible"] = filter.Responsible
	}
	if filter.Type != "" {
		query["type"] = filter.Type
	}
	if filter.Location != "" {
		query["location"] = filter.Location
	}

	if filter.Search != "" {
		searchQuery := []bson.M{
			{"name": bson.M{"$regex": filter.Search, "$options": "i"}},
			{"code": bson.M{"$regex": filter.Search, "$options": "i"}},
			{"responsible": bson.M{"$regex": filter.Search, "$options": "i"}},
		}

		if len(query) == 0 {
			query["$or"] = searchQuery
		} else {
			query = bson.M{
				"$and": []bson.M{
					query,
					{"$or": searchQuery},
				},
			}
		}
	}

	return query
}
