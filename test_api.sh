#!/bin/bash

echo "Testing text storage (default):"
curl -X POST "http://localhost:8080/" \
  -H "Content-Type: application/json" \
  -d '{"ttl":"1d","data":"Hello world, this is a test!"}'
echo -e "\n"

echo "Testing URL storage:"
curl -X POST "http://localhost:8080/" \
  -H "Content-Type: application/json" \
  -d '{"ttl":"1d","data":"https://example.com","typ":"URL"}'
echo -e "\n"

echo "Testing form submission for text:"
curl -X POST "http://localhost:8080/?input=urlencoded" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "ttl=1d&data=Hello+from+form&typ=Text"
echo -e "\n"

echo "Testing form submission for URL:"
curl -X POST "http://localhost:8080/?input=urlencoded" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "ttl=1d&data=https%3A%2F%2Fexample.com&typ=URL"
echo -e "\n"