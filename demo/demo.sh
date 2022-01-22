#!/bin/sh

white=`tput setaf 7`
red=`tput setaf 1`
green=`tput setaf 2`


echo "mangtas word count service demo\n"

echo "first demo:$green empty input\n$white"

out=$(curl -X POST -s 127.0.0.1:8080/v1/addwords)

echo "output: $red$out\n$white"

input='b*d,w0RD$'

echo "second demo:$green adding bad words: $input\n$white" 

out=$(curl -X POST -s 127.0.0.1:8080/v1/addwords -d ""$input)

echo "output: $red$out\n$white"

input="good,words,"$input

echo "third demo:$green adding a mix of good and bad words: $input\n$white" 

out=$(curl -X POST -s 127.0.0.1:8080/v1/addwords -d ""$input)

echo "output: $red$out\n$white"

input=$(cat words.txt)

echo "fourth demo:$green adding valid words:\n$input\n$white" 

# out=$(curl -X POST -s 127.0.0.1:8080/v1/addwords -d $input)
out=$(curl -X POST -s 127.0.0.1:8080/v1/addwords -d @./words.txt)

echo "output: $red$out\n$white"

echo "final demo:$green listing top 10 added words\n$white" 

out=$(curl -X GET -s 127.0.0.1:8080/v1/gettopwords)

echo "output:"  
echo $out | jq
