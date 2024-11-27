# markdown conversion

This project aims to build a renderer that produces a js script to render a markdown file in a browser.  

## simple conversion to html

'simpleMdCon'  
program that reads a markdown file and writes a html file.  

## conversion with the md2js renderer

'simpleMd2JsConv'  
A program that reads a markdown file and produces a js output file.  
The js output file renders a html page in the browser.  

status: in testing  

## conversion with the md2jsV2 renderer

'simpleMd2JsConvV2'  
 - adds meta data contained in a yaml file
 - adds style objects located in the style js file to style the output

status: in progress  

## md2jsV3: Performance enhancement

replaced rendering textblocks and paragraphs that have multiple inline 
renderering functions with a single renderTextChildren function

status: in testing  

## AstDump

dumps the ast tree of a document to a text file.  

status: working
