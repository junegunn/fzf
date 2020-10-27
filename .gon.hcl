source = ["./dist/fzf-macos_darwin_amd64/fzf"]
bundle_id = "kr.junegunn.fzf"

apple_id {
  username = "junegunn.c@gmail.com"
  password = "@env:AC_PASSWORD"
}

sign {
  application_identity = "Apple Development: junegunn.c@gmail.com"
}
