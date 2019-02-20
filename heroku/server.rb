# A tiny server using the Heroku stack's built-in Ruby.

# This Ruby language file is part of the test
# `TestAccHerokuBuild_LocalSourceDirectorySelfContained`
# in `resource_heroku_build_test.go` 

require 'webrick'

server = WEBrick::HTTPServer.new :Port => ENV["PORT"]

server.mount_proc '/' do |req, res|
    res.body = "Hello, world!\n"
end

trap 'INT' do
  server.shutdown
end

server.start
