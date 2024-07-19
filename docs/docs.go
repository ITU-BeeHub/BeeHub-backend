// Package docs Code generated by swaggo/swag. DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {},
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/auth/login": {
            "post": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Login"
                ],
                "summary": "Hello World",
                "parameters": [
                    {
                        "description": "Login credentials",
                        "name": "login",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/auth.LoginRequest"
                        }
                    }
                ],
                "responses": {}
            }
        },
        "/auth/profile": {
            "get": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Profile"
                ],
                "summary": "Hello World",
                "responses": {}
            }
        },
        "/beePicker/courses": {
            "get": {
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "BeePicker"
                ],
                "summary": "Retrieves courses from the BeePicker.",
                "responses": {}
            }
        },
        "/beePicker/schedule": {
            "get": {
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "BeePicker"
                ],
                "summary": "Retrieves schedules from the BeePicker.",
                "responses": {}
            },
            "post": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "BeePicker"
                ],
                "summary": "Selects a course from the BeePicker.",
                "parameters": [
                    {
                        "description": "Request body containing the ECRN",
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/beepicker.ScheduleSaveRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Selection successful",
                        "schema": {
                            "type": "string"
                        }
                    },
                    "400": {
                        "description": "Bad request",
                        "schema": {
                            "type": "string"
                        }
                    },
                    "500": {
                        "description": "Internal server error",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        },
        "/hello": {
            "get": {
                "description": "Hello World",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Hello"
                ],
                "summary": "Hello World",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/main.MessageResponse"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "auth.LoginRequest": {
            "type": "object",
            "required": [
                "email",
                "password"
            ],
            "properties": {
                "email": {
                    "type": "string"
                },
                "password": {
                    "type": "string"
                }
            }
        },
        "beepicker.ScheduleSaveRequest": {
            "type": "object",
            "properties": {
                "ECRN": {
                    "type": "array",
                    "items": {
                        "type": "integer"
                    }
                },
                "SCRN": {
                    "type": "array",
                    "items": {
                        "type": "integer"
                    }
                },
                "scheduleName": {
                    "type": "string"
                }
            }
        },
        "main.MessageResponse": {
            "type": "object",
            "properties": {
                "message": {
                    "type": "string"
                }
            }
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "localhost:8080",
	BasePath:         "/",
	Schemes:          []string{},
	Title:            "BeeHub Ders Seçim Botu API",
	Description:      "Bu, BeeHub Ders Seçim Botu için API dokümantasyonudur.",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
