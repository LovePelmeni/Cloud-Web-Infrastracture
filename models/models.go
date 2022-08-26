package models

import (
	"fmt"
	"log"

	"os"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	DebugLogger *log.Logger
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

var (
	Database *gorm.DB
)

var (
	DATABASE_NAME     = os.Getenv("DATABASE_NAME")
	DATABASE_HOST     = os.Getenv("DATABASE_HOST")
	DATABASE_PORT     = os.Getenv("DATABASE_PORT")
	DATABASE_USER     = os.Getenv("DATABASE_USER")
	DATABASE_PASSWORD = os.Getenv("DATABASE_PASSWORD")
)

func init() {
	DatabaseInstance, ConnectionError := gorm.Open(postgres.New(postgres.Config{
		DSN: fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s",
			DATABASE_HOST, DATABASE_PORT, DATABASE_USER, DATABASE_PASSWORD, DATABASE_NAME),
	}))

	switch ConnectionError {

	case gorm.ErrInvalidDB:
		panic("Please Setup Credentials for your PostgreSQL Database: Host, Port, User, Password, DbName")

	case gorm.ErrUnsupportedDriver:
		panic("Invalid Database Driver")

	case gorm.ErrNotImplemented:
		panic(ConnectionError)
	}

	Database = DatabaseInstance
	Database.AutoMigrate(&SSHPublicKey{}, &Customer{}, &VirtualMachine{})
	LogFile, Error := os.OpenFile("Models.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	DebugLogger = log.New(LogFile, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	InfoLogger = log.New(LogFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(LogFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	if Error != nil {
		panic(Error)
	}
}

type Customer struct {
	gorm.Model
	Username string `json:"Username" gorm:"<-:create;type:varchar(100); not null; unique;"`
	Email    string `json:"Email" gorm:"<-:create;type:varchar(100); not null; unique;"`
	Password string `json:"Password" gorm:"type:varchar(100); not null;"`
}

func NewCustomer(Username string, Password string, Email string) *Customer {
	PasswordHash, HashError := bcrypt.GenerateFromPassword([]byte(Password), 14)
	if HashError != nil {
		return nil
	}
	return &Customer{
		Username: Username,
		Email:    Email,
		Password: string(PasswordHash),
	}
}

func (this *Customer) Create() (*gorm.DB, error) {
	// Creates New Customer Profile
	CreatedCustomer := Database.Model(&Customer{}).Create(this)
	return CreatedCustomer, CreatedCustomer.Error
}

func (this *Customer) Delete() (*gorm.DB, error) {
	// Deletes Customer Profile
	DeletedCustomer := Database.Model(&Customer{}).Delete(&this)
	Database.Model(&Customer{}).Unscoped().Delete(&this)
	return DeletedCustomer, DeletedCustomer.Error
}

type VirtualMachine struct {
	gorm.Model

	OwnerId            string `json:"OwnerId" gorm:"<-:create;type:varchar(100);not null;unique;"`
	VirtualMachineName string `json:"VirtualMachineName" gorm:"type:varchar(100);not null;"`
	ItemPath           string `json:"ItemPath" gorm:"<-:create;type:varchar(100);not null;"`
	IPAddress          string `json:"IPAddress" gorm:"<-:create;type:varchar(100);not null;unique;"`
}

func NewVirtualMachine(

	OwnerId string, // ID Of the Customer, who Owns this Virtual Machine
	VirtualMachineName string, // Virtual Machine UniqueName
	ItemPath string,
	IPAddress string,

) *VirtualMachine {

	return &VirtualMachine{
		OwnerId:            OwnerId,
		VirtualMachineName: VirtualMachineName,
		ItemPath:           ItemPath,
		IPAddress:          IPAddress,
	}
}

func (this *VirtualMachine) Create() (*gorm.DB, error) {
	// Creates New Virtual Machine Object
	Created := Database.Clauses(clause.OnConflict{Columns: []clause.Column{
		{Table: "VirtualMachine", Name: "VirtualMachineName"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"VirtualMachineName": gorm.Expr("virtual_machine_name + _ + uuid_generate_v3()"),
		})}).Model(&VirtualMachine{}).Create(&this)
	return Created, Created.Error
}

func (this *VirtualMachine) Delete() (*gorm.DB, error) {
	// Deletes the Virtual Machine ORM Object....
	Deleted := Database.Clauses(clause.OnConflict{DoNothing: true}).Delete(&this)
	Database.Model(&VirtualMachine{}).Unscoped().Model(&VirtualMachine{}).Delete(&this)
	return Deleted, Deleted.Error
}

type SSHPublicKey struct {
	gorm.Model
	Key              string `json:"Key" xml:"Key" gorm:"type:varchar(10000);default:null;"`
	Filename         string `json:"Filename" xml:"Filename" gorm:"type:varchar(100); not null;"`
	VirtualMachineID int    `json:"VirtualMachineID" xml:"VirtualMachineID" gorm:"unique;primaryKey;"`
}

func NewSshPublicKey(KeyContent []byte, Filename string) *SSHPublicKey {
	return &SSHPublicKey{
		Key:      string(KeyContent),
		Filename: Filename,
	}
}

func (this *SSHPublicKey) Create() (*gorm.DB, error) {
	// Creates New Record of the SSHPublicKey Model at the Database
	newSSHKey := Database.Model(&SSHPublicKey{}).Create(&this)
	return newSSHKey, newSSHKey.Error
}

func (this *SSHPublicKey) Update(NewSshKey []byte, Filename ...string) (*gorm.DB, error) {
	// Updates SSH Keys Locally at the Database
	Gorm := Database.Model(&SSHPublicKey{}).Unscoped().Where(
		"virtual_machine_id = ?", this.VirtualMachineID).Update("Key", NewSshKey)
	return Gorm, Gorm.Error
}

func (this *SSHPublicKey) Delete() (*gorm.DB, error) {
	// Deletes Public SSH Key Related to the Virtual Machine
	Deleted := Database.Clauses(clause.OnConflict{Columns: []clause.Column{
		{
			Name:  "virtual_machine_id", // Checking if ForeignKey Constraint is not Throwing An Error
			Table: "SSHPublicKey",
		},
		{
			Name:  "virtual_machine",
			Table: "SSHPublicKey",
		},
	}, DoUpdates: clause.AssignmentColumns([]string{})}).Model(&SSHPublicKey{}).Delete(&this)
	Database.Model(&SSHPublicKey{}).Unscoped().Delete(&this)
	return Deleted, Deleted.Error
}
