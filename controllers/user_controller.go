package controllers

import (
	"context"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/gergpol1998/gin-mongo-api/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type BaseController struct {
	Collection *mongo.Collection
}

func NewBaseController(collection *mongo.Collection) *BaseController {
	return &BaseController{Collection: collection}
}

func isValidEmail(email string) bool {
	emailPattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,4}$`
	return regexp.MustCompile(emailPattern).MatchString(email)
}

func validateAge(age int) bool {
	return age >= 1 && age <= 100
}

func handleError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

func (bc *BaseController) saveUploadedFile(c *gin.Context, file *multipart.FileHeader, uploadDir string) error {
	err := c.SaveUploadedFile(file, filepath.Join(uploadDir, file.Filename))
	if err != nil {
		return err
	}
	return nil
}

type UserController struct {
	*BaseController
}

func NewUserController(collection *mongo.Collection) *UserController {
	return &UserController{BaseController: NewBaseController(collection)}
}

// create data
func (uc *UserController) Create(c *gin.Context) {
	var user models.User

	user.Name = c.PostForm("name")
	ageStr := c.PostForm("age")
	if ageStr == "" {
		user.Age = 0 // Set age to 0 if not provided
	} else {
		age, err := strconv.Atoi(ageStr)
		if err != nil {
			handleError(c, http.StatusUnauthorized, "Invalid age value")
			return
		}

		if !validateAge(age) {
			handleError(c, http.StatusUnauthorized, "Please enter an age not less than 1 or not more than 100")
			return
		}

		user.YearOfBirth = time.Now().Year() - age
		user.Age = age
	}
	user.Note = c.PostForm("note")
	user.Email = c.PostForm("email")
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	if user.Name == "" || user.Age == 0 || user.Email == "" {
		handleError(c, http.StatusUnauthorized, "Please fill out all required fields")
		return
	}

	if !isValidEmail(user.Email) {
		handleError(c, http.StatusUnauthorized, "Invalid email format")
		return
	}

	existingEmail := models.User{}
	err := uc.Collection.FindOne(context.Background(), bson.M{"email": user.Email}).Decode(&existingEmail)
	if err == nil {
		handleError(c, http.StatusUnauthorized, "Email already exists")
		return
	} else if err != mongo.ErrNoDocuments {
		handleError(c, http.StatusInternalServerError, "Failed to check email uniqueness")
		return
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		handleError(c, http.StatusUnauthorized, err.Error())
		return
	}

	ext := filepath.Ext(file.Filename)
	if ext != ".jpg" && ext != ".png" {
		handleError(c, http.StatusUnauthorized, "Invalid file format. Only JPG and PNG allowed.")
		return
	}

	avatarName := file.Filename
	avatarType := ext[1:]

	user.AvatarName = avatarName
	user.AvatarType = avatarType

	_, err = uc.Collection.InsertOne(context.Background(), user)
	if err != nil {
		handleError(c, http.StatusInternalServerError, "Failed to create user")
		return
	}

	err = uc.saveUploadedFile(c, file, "uploads")
	if err != nil {
		handleError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, user)
}

// update data
func (uc *UserController) Update(c *gin.Context) {
	var user models.User

	userID := c.Param("user_id")
	if userID == "" {
		handleError(c, http.StatusUnauthorized, "User ID is required")
		return
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		handleError(c, http.StatusUnauthorized, "Invalid User ID format")
		return
	}

	existingUser := models.User{}
	err = uc.Collection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&existingUser)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			handleError(c, http.StatusUnauthorized, "User not found")
			return
		}
		handleError(c, http.StatusInternalServerError, "Failed to fetch user data")
		return
	}

	// Update the user properties if provided
	if name := c.PostForm("name"); name != "" {
		user.Name = name
	}

	ageStr := c.PostForm("age")
	if ageStr == "" {
		user.Age = 0 // Set age to 0 if not provided
	} else {
		age, err := strconv.Atoi(ageStr)
		if err != nil {
			handleError(c, http.StatusUnauthorized, "Invalid age value")
			return
		}

		if !validateAge(age) {
			handleError(c, http.StatusUnauthorized, "Please enter an age not less than 1 or not more than 100")
			return
		}

		user.YearOfBirth = time.Now().Year() - age
		user.Age = age
	}

	if note := c.PostForm("note"); note != "" {
		if note != "clean" {
			user.Note = note
		} else {
			remove := bson.M{"$unset": bson.M{"note": ""}}
			_, err := uc.Collection.UpdateOne(
				context.Background(),
				bson.M{"_id": objectID},
				remove,
			)
			if err != nil {
				handleError(c, http.StatusInternalServerError, "Failed to remove note field")
				return
			}
		}
	}

	if email := c.PostForm("email"); email != "" {
		if !isValidEmail(user.Email) {
			handleError(c, http.StatusUnauthorized, "Invalid email format")
			return
		}

		existingEmail := models.User{}
		err = uc.Collection.FindOne(context.Background(), bson.M{"email": user.Email}).Decode(&existingEmail)
		if err == nil {
			handleError(c, http.StatusUnauthorized, "Email already exists")
			return
		} else if err != mongo.ErrNoDocuments {
			handleError(c, http.StatusInternalServerError, "Failed to check email uniqueness")
			return
		}
	}

	// Handle avatar file upload if provided
	file, err := c.FormFile("avatar")
	if err != nil && err != http.ErrMissingFile {
		handleError(c, http.StatusUnauthorized, err.Error())
		return
	}

	if file != nil {
		ext := filepath.Ext(file.Filename)
		if ext != ".jpg" && ext != ".png" {
			handleError(c, http.StatusUnauthorized, "Invalid file format. Only JPG and PNG allowed.")
			return
		}

		avatarName := file.Filename
		avatarType := ext[1:]

		user.AvatarName = avatarName
		user.AvatarType = avatarType

		err = uc.saveUploadedFile(c, file, "uploads")
		if err != nil {
			handleError(c, http.StatusInternalServerError, err.Error())
			return
		}
	}

	user.UpdatedAt = time.Now()

	_, err = uc.Collection.UpdateOne(
		context.Background(),
		bson.M{"_id": objectID},
		bson.M{"$set": user},
	)

	if err != nil {
		handleError(c, http.StatusInternalServerError, "Failed to update user data")
		return
	}

	c.JSON(http.StatusOK, user)
}

// list users with pagination and sorting
func (uc *UserController) List(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	pageStr := c.DefaultQuery("page", "1")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		handleError(c, http.StatusUnauthorized, "Invalid limit value")
		return
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil {
		handleError(c, http.StatusUnauthorized, "Invalid page value")
		return
	}

	if limit <= 0 || page <= 0 {
		handleError(c, http.StatusUnauthorized, "Limit and page values must be positive")
		return
	}

	skip := (page - 1) * limit

	findOptions := options.Find()
	findOptions.SetLimit(int64(limit)).SetSkip(int64(skip))
	findOptions.SetSort(bson.D{{Key: "created_at", Value: -1}}) // Sort by created_at in descending order

	cursor, err := uc.Collection.Find(context.Background(), bson.D{}, findOptions)
	if err != nil {
		handleError(c, http.StatusInternalServerError, "Failed to fetch user data")
		return
	}
	defer cursor.Close(context.Background())

	var users []models.User
	if err := cursor.All(context.Background(), &users); err != nil {
		handleError(c, http.StatusInternalServerError, "Failed to decode user data")
		return
	}

	// Count the total number of users
	totalCount, err := uc.Collection.CountDocuments(context.Background(), bson.D{})
	if err != nil {
		handleError(c, http.StatusInternalServerError, "Failed to count users")
		return
	}

	response := map[string]interface{}{
		"count": totalCount,
		"data":  users,
	}

	c.JSON(http.StatusOK, response)
}

// get user by user_id
func (uc *UserController) GetByID(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		handleError(c, http.StatusUnauthorized, "User ID is required")
		return
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		handleError(c, http.StatusUnauthorized, "Invalid User ID format")
		return
	}

	user := models.User{}
	err = uc.Collection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			handleError(c, http.StatusNotFound, "User not found")
			return
		}
		handleError(c, http.StatusInternalServerError, "Failed to fetch user data")
		return
	}

	c.JSON(http.StatusOK, user)
}

// delete user by user_id
func (uc *UserController) DeleteByID(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		handleError(c, http.StatusUnauthorized, "User ID is required")
		return
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		handleError(c, http.StatusUnauthorized, "Invalid User ID format")
		return
	}

	result, err := uc.Collection.DeleteOne(context.Background(), bson.M{"_id": objectID})
	if err != nil {
		handleError(c, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	if result.DeletedCount == 0 {
		handleError(c, http.StatusNotFound, "User not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
