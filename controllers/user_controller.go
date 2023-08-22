// package controllers คือแพคเกจที่เก็บความเกี่ยวข้องกับการควบคุมและจัดการข้อมูลผู้ใช้
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

// BaseController คือตัวควบคุมหลักที่มีความสามารถในการจัดการข้อมูลใน MongoDB Collection อาทิเช่นการเพิ่ม แก้ไข ลบข้อมูล
type BaseController struct {
	Collection *mongo.Collection
}

// NewBaseController สร้างและคืนค่า BaseController ใหม่ที่เชื่อมต่อกับ Collection ที่กำหนด
func NewBaseController(collection *mongo.Collection) *BaseController {
	return &BaseController{Collection: collection}
}

// isValidEmail ตรวจสอบว่ารูปแบบของอีเมลถูกต้องหรือไม่
func isValidEmail(email string) bool {
	emailPattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,4}$`
	return regexp.MustCompile(emailPattern).MatchString(email)
}

// validateAge ตรวจสอบความถูกต้องของอายุ
func validateAge(age int) bool {
	return age >= 1 && age <= 100
}

// handleError สร้างและส่งคืน JSON response แสดงข้อความผิดพลาดพร้อมรหัสสถานะที่กำหนด
func handleError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

// saveUploadedFile บันทึกไฟล์ที่อัปโหลดลงในไดเรกทอรี uploadDir
func (bc *BaseController) saveUploadedFile(c *gin.Context, file *multipart.FileHeader, uploadDir string) error {
	err := c.SaveUploadedFile(file, filepath.Join(uploadDir, file.Filename))
	if err != nil {
		return err
	}
	return nil
}

// UserController คือตัวควบคุมที่สืบทอดมาจาก BaseController และมีฟังก์ชันสำหรับการจัดการข้อมูลผู้ใช้
type UserController struct {
	*BaseController
}

// NewUserController สร้างและคืนค่า UserController ใหม่ที่เชื่อมต่อกับ Collection ที่กำหนด
func NewUserController(collection *mongo.Collection) *UserController {
	return &UserController{BaseController: NewBaseController(collection)}
}

// Create เป็นฟังก์ชันที่ใช้ในการสร้างข้อมูลผู้ใช้ใหม่
func (uc *UserController) Create(c *gin.Context) {
	var user models.User

	// รับข้อมูลจาก PostForm
	user.Name = c.PostForm("name")
	ageStr := c.PostForm("age")
	if ageStr == "" {
		user.Age = 0 // ตั้งค่าอายุเป็น 0 หากไม่ได้ระบุ
	} else {
		age, err := strconv.Atoi(ageStr)
		if err != nil {
			handleError(c, http.StatusUnauthorized, "อายุไม่ถูกต้อง")
			return
		}

		if !validateAge(age) {
			handleError(c, http.StatusUnauthorized, "กรุณากรอกอายุระหว่าง 1 ถึง 100")
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
		handleError(c, http.StatusUnauthorized, "กรุณากรอกข้อมูลที่จำเป็นให้ครบถ้วน")
		return
	}

	if !isValidEmail(user.Email) {
		handleError(c, http.StatusUnauthorized, "รูปแบบอีเมลไม่ถูกต้อง")
		return
	}

	// ตรวจสอบความเป็นเอกลักษณ์ของอีเมล
	existingEmail := models.User{}
	err := uc.Collection.FindOne(context.Background(), bson.M{"email": user.Email}).Decode(&existingEmail)
	if err == nil {
		handleError(c, http.StatusUnauthorized, "อีเมลนี้มีอยู่แล้ว")
		return
	} else if err != mongo.ErrNoDocuments {
		handleError(c, http.StatusInternalServerError, "เกิดข้อผิดพลาดในการตรวจสอบความเป็นเอกลักษณ์ของอีเมล")
		return
	}

	// อัปโหลดไฟล์รูป Avatar
	file, err := c.FormFile("avatar")
	if err != nil {
		handleError(c, http.StatusUnauthorized, err.Error())
		return
	}

	ext := filepath.Ext(file.Filename)
	if ext != ".jpg" && ext != ".png" {
		handleError(c, http.StatusUnauthorized, "รูปแบบไฟล์ไม่ถูกต้อง เฉพาะ JPG และ PNG เท่านั้น")
		return
	}

	avatarName := file.Filename
	avatarType := ext[1:]

	user.AvatarName = avatarName
	user.AvatarType = avatarType

	// บันทึกข้อมูลผู้ใช้ลงใน Collection
	_, err = uc.Collection.InsertOne(context.Background(), user)
	if err != nil {
		handleError(c, http.StatusInternalServerError, "ไม่สามารถสร้างผู้ใช้ได้")
		return
	}

	// บันทึกไฟล์รูป Avatar
	err = uc.saveUploadedFile(c, file, "uploads")
	if err != nil {
		handleError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, user)
}

// Update เป็นฟังก์ชันที่ใช้ในการอัปเดตข้อมูลผู้ใช้
func (uc *UserController) Update(c *gin.Context) {
	var user models.User

	// รับค่า user_id จาก URL Parameter
	userID := c.Param("user_id")
	if userID == "" {
		handleError(c, http.StatusUnauthorized, "จำเป็นต้องระบุ User ID")
		return
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		handleError(c, http.StatusUnauthorized, "รูปแบบ User ID ไม่ถูกต้อง")
		return
	}

	// ค้นหาข้อมูลผู้ใช้ที่มี user_id ที่ระบุ
	existingUser := models.User{}
	err = uc.Collection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&existingUser)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			handleError(c, http.StatusUnauthorized, "ไม่พบผู้ใช้")
			return
		}
		handleError(c, http.StatusInternalServerError, "เกิดข้อผิดพลาดในการเรียกดูข้อมูลผู้ใช้")
		return
	}

	// อัปเดตคุณสมบัติของผู้ใช้หากมีการระบุ
	if name := c.PostForm("name"); name != "" {
		user.Name = name
	}

	// รับข้อมูลอายุจาก PostForm
	ageStr := c.PostForm("age")
	if ageStr == "" {
		user.Age = 0 // ตั้งค่าอายุเป็น 0 หากไม่ได้ระบุ
	} else {
		age, err := strconv.Atoi(ageStr)
		if err != nil {
			handleError(c, http.StatusUnauthorized, "อายุไม่ถูกต้อง")
			return
		}

		if !validateAge(age) {
			handleError(c, http.StatusUnauthorized, "กรุณากรอกอายุระหว่าง 1 ถึง 100")
			return
		}

		user.YearOfBirth = time.Now().Year() - age
		user.Age = age
	}

	// ตรวจสอบและจัดการเรื่องของข้อมูล Note
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
				handleError(c, http.StatusInternalServerError, "ไม่สามารถลบฟิลด์ Note ได้")
				return
			}
		}
	}

	// ตรวจสอบและจัดการข้อมูลอีเมล
	if email := c.PostForm("email"); email != "" {
		if !isValidEmail(user.Email) {
			handleError(c, http.StatusUnauthorized, "รูปแบบอีเมลไม่ถูกต้อง")
			return
		}

		// ตรวจสอบความเป็นเอกลักษณ์ของอีเมล
		existingEmail := models.User{}
		err = uc.Collection.FindOne(context.Background(), bson.M{"email": user.Email}).Decode(&existingEmail)
		if err == nil {
			handleError(c, http.StatusUnauthorized, "อีเมลนี้มีอยู่แล้ว")
			return
		} else if err != mongo.ErrNoDocuments {
			handleError(c, http.StatusInternalServerError, "เกิดข้อผิดพลาดในการตรวจสอบความเป็นเอกลักษณ์ของอีเมล")
			return
		}
	}

	// ตรวจสอบและจัดการไฟล์รูป Avatar หากมีการอัปโหลด
	file, err := c.FormFile("avatar")
	if err != nil && err != http.ErrMissingFile {
		handleError(c, http.StatusUnauthorized, err.Error())
		return
	}

	if file != nil {
		ext := filepath.Ext(file.Filename)
		if ext != ".jpg" && ext != ".png" {
			handleError(c, http.StatusUnauthorized, "รูปแบบไฟล์ไม่ถูกต้อง เฉพาะ JPG และ PNG เท่านั้น")
			return
		}

		avatarName := file.Filename
		avatarType := ext[1:]

		user.AvatarName = avatarName
		user.AvatarType = avatarType

		// บันทึกไฟล์รูป Avatar
		err = uc.saveUploadedFile(c, file, "uploads")
		if err != nil {
			handleError(c, http.StatusInternalServerError, err.Error())
			return
		}
	}

	user.UpdatedAt = time.Now()

	// อัปเดตข้อมูลผู้ใช้ใน Collection
	_, err = uc.Collection.UpdateOne(
		context.Background(),
		bson.M{"_id": objectID},
		bson.M{"$set": user},
	)

	if err != nil {
		handleError(c, http.StatusInternalServerError, "ไม่สามารถอัปเดตข้อมูลผู้ใช้ได้")
		return
	}

	c.JSON(http.StatusOK, user)
}

// List เป็นฟังก์ชันที่ใช้ในการแสดงรายการผู้ใช้พร้อมกำหนดการแบ่งหน้าและการเรียงลำดับ
func (uc *UserController) List(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	pageStr := c.DefaultQuery("page", "1")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		handleError(c, http.StatusUnauthorized, "ค่า limit ไม่ถูกต้อง")
		return
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil {
		handleError(c, http.StatusUnauthorized, "ค่า page ไม่ถูกต้อง")
		return
	}

	if limit <= 0 || page <= 0 {
		handleError(c, http.StatusUnauthorized, "ค่า limit และ page ต้องเป็นบวก")
		return
	}

	skip := (page - 1) * limit

	// กำหนดตัวเลือกในการค้นหา แบ่งหน้า และเรียงลำดับ
	findOptions := options.Find()
	findOptions.SetLimit(int64(limit)).SetSkip(int64(skip))
	findOptions.SetSort(bson.D{{Key: "created_at", Value: -1}}) // เรียงตาม created_at จากใหม่ไปเก่า

	// ค้นหาผู้ใช้และดึงข้อมูล
	cursor, err := uc.Collection.Find(context.Background(), bson.D{}, findOptions)
	if err != nil {
		handleError(c, http.StatusInternalServerError, "ไม่สามารถดึงข้อมูลผู้ใช้ได้")
		return
	}
	defer cursor.Close(context.Background())

	// ดึงข้อมูลผู้ใช้ทั้งหมดใน Slice users
	var users []models.User
	if err := cursor.All(context.Background(), &users); err != nil {
		handleError(c, http.StatusInternalServerError, "ไม่สามารถแปลงข้อมูลผู้ใช้ได้")
		return
	}

	// นับจำนวนผู้ใช้ทั้งหมด
	totalCount, err := uc.Collection.CountDocuments(context.Background(), bson.D{})
	if err != nil {
		handleError(c, http.StatusInternalServerError, "ไม่สามารถนับจำนวนผู้ใช้ได้")
		return
	}

	// สร้าง JSON response
	response := map[string]interface{}{
		"count": totalCount,
		"data":  users,
	}

	c.JSON(http.StatusOK, response)
}

// GetByID เป็นฟังก์ชันที่ใช้ในการแสดงข้อมูลผู้ใช้ตาม user_id
func (uc *UserController) GetByID(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		handleError(c, http.StatusUnauthorized, "จำเป็นต้องระบุ User ID")
		return
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		handleError(c, http.StatusUnauthorized, "รูปแบบ User ID ไม่ถูกต้อง")
		return
	}

	user := models.User{}
	err = uc.Collection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			handleError(c, http.StatusNotFound, "ไม่พบผู้ใช้")
			return
		}
		handleError(c, http.StatusInternalServerError, "ไม่สามารถดึงข้อมูลผู้ใช้ได้")
		return
	}

	c.JSON(http.StatusOK, user)
}

// DeleteByID เป็นฟังก์ชันที่ใช้ในการลบข้อมูลผู้ใช้ตาม user_id
func (uc *UserController) DeleteByID(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		handleError(c, http.StatusUnauthorized, "จำเป็นต้องระบุ User ID")
		return
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		handleError(c, http.StatusUnauthorized, "รูปแบบ User ID ไม่ถูกต้อง")
		return
	}

	// ลบข้อมูลผู้ใช้จาก Collection
	result, err := uc.Collection.DeleteOne(context.Background(), bson.M{"_id": objectID})
	if err != nil {
		handleError(c, http.StatusInternalServerError, "ไม่สามารถลบผู้ใช้ได้")
		return
	}

	if result.DeletedCount == 0 {
		handleError(c, http.StatusNotFound, "ไม่พบผู้ใช้")
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
