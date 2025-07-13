#!/bin/bash

echo "Testing text storage (default):"
curl -X POST "http://localhost:8080/" \
  -H "Content-Type: text/plain" \
  -d 'Hello world, this is a test!'
echo -e "\n"

echo "Testing pastebin storage with large content:"
curl -X POST "http://localhost:8080/" \
  -H "Content-Type: text/plain" \
  -d 'This is a longer text that would be stored as a pastebin entry.'
echo -e "\n"

echo "Testing form submission:"
curl -X POST "http://localhost:8080/?input=form" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "data=Hello+from+form"
echo -e "\n"

echo "Testing form submission with encoded content:"
curl -X POST "http://localhost:8080/?input=form" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "data=Hello%20world%20with%20special%20characters%21"
echo -e "\n"