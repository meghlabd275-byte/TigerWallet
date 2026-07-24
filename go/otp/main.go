package main
import("log";"net/http")
func main(){http.HandleFunc("/",func(w http.ResponseWriter,r *http.Request){});log.Println("OTP on :8104");http.ListenAndServe(":8104",nil)}
