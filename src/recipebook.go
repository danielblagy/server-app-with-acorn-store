package main


import (
	"fmt"
	"github.com/danielblagy/acorn-store-api-golang"
	"github.com/tidwall/sjson"
	"github.com/tidwall/gjson"
	"strconv"
	"net/http"
	"html/template"
)


var (
	db *AcornStore.Db
	idCounter int64
)


type Recipe struct {
	
	Title string
	Body string
}

func (r *Recipe) save() error {
	
	recipeDocument, _ := sjson.Set("", "id", idCounter)
	recipeDocument, _ = sjson.Set(recipeDocument, "title", r.Title)
	recipeDocument, _ = sjson.Set(recipeDocument, "body", r.Body)
	
	return db.Collection("recipes").Insert(recipeDocument)
}

func loadRecipe(title string) (*Recipe, error) {
	
	fmt.Printf("load recipe %v\n", title)
	object, err := db.Collection("recipes").Retrieve(".#(title=\""+title+"\")#")
	fmt.Printf("received '%v'\n", object)
	if err != nil {
		return &Recipe{Title: "", Body: ""}, err
	}
	
	body := gjson.Get(object, "0.body")
	if !body.Exists() {
		return &Recipe{Title: "", Body: ""}, fmt.Errorf("no recipe found")
	}
	
	return &Recipe{Title: title, Body: body.String()}, nil
}


var templates = template.Must(template.ParseFiles("src/view.html", "src/edit.html"))

func renderHTMLTemplate(writer http.ResponseWriter, template_name string, recipe *Recipe) {
    template_path := template_name + ".html"
	
    err := templates.ExecuteTemplate(writer, template_path, recipe)
    if err != nil {
        http.Error(writer, err.Error(), http.StatusInternalServerError)
    }
}

func viewHandler(writer http.ResponseWriter, request *http.Request) {
	
	title := request.URL.Path[len("/view/"):]
	
	recipe, err := loadRecipe(title)
	if err != nil {
		// if no such recipe exists, redirect to the edit page (for user to create a new recipe)
		http.Redirect(writer, request, "/edit/" + title, http.StatusFound)
        return
	}
	
	renderHTMLTemplate(writer, "view", recipe)
}

func editHandler(writer http.ResponseWriter, request *http.Request) {
	
	title := request.URL.Path[len("/edit/"):]
	
	recipe, err := loadRecipe(title)
	if err != nil {
		recipe = &Recipe{Title: title}
	}
	
	renderHTMLTemplate(writer, "edit", recipe)
}

func saveHandler(writer http.ResponseWriter, request *http.Request) {
    title := request.URL.Path[len("/save/"):]
    
	body := request.FormValue("body")
	
	// check if we need to simply update
	recipe, err := loadRecipe(title)
	if err == nil {
		fmt.Printf("updating %v\n", recipe)
		db.Collection("recipes").Update(".#(title=\""+title+"\")#", "body", "\""+body+"\"")
	} else {
		recipe := &Recipe{Title: title, Body: body}
    	if err := recipe.save(); err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	
    http.Redirect(writer, request, "/view/" + title, http.StatusFound)
}

func deleteHandler(writer http.ResponseWriter, request *http.Request) {
	
	title := request.URL.Path[len("/delete/"):]
	
	recipe, err := loadRecipe(title)
	if err != nil {
		// if no such recipe exists, redirect to the edit page (for user to create a new recipe)
		http.Redirect(writer, request, "/edit/" + title, http.StatusFound)
        return
	}
	
	if deleteErr := db.Collection("recipes").Delete(".#(title=\""+recipe.Title+"\")#"); deleteErr != nil {
		renderHTMLTemplate(writer, "view", &Recipe{Title: "failed to delete", Body: ""})
	} else {
		renderHTMLTemplate(writer, "view", &Recipe{Title: "recipe deleted", Body: ""})
	}
}


func main() {
	
	dbconn, err := AcornStore.Connect("acorn-store://localhost:2525/recipe-webapp/root:1234")
	if err != nil {
		fmt.Printf("Failed to connect to the db\n")
	}
	
	db = &dbconn
	
	size, err := db.Collection("recipes").Retrieve(".#")
	if err != nil {
		fmt.Printf("Failed to get the size of recipes collection\n")
	}
	
	idCounter, _ = strconv.ParseInt(size, 10, 0)
	
	fmt.Printf("idCounter: %v\n", idCounter)
	
	http.HandleFunc("/view/", viewHandler)
	http.HandleFunc("/edit/", editHandler)
	http.HandleFunc("/save/", saveHandler)
	http.HandleFunc("/delete/", deleteHandler)
	
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println(err)
	}
}