# markdown conversion

This project aims to build a renderer that produces a js script to render a markdown file in a browser.  

## simple conversion to html

_simpleMdCon_  
program that reads a markdown file and writes a html file.  


## conversion to js Dom with the md2jsV3 renderer

_ConvMd2JsV3_  
A program that reads a markdown file and produces a js script file that can render the md file in a browser.  
There is a start file, that wraps the converted js script into a js function.
There is a style file, that adds styling to the md output.

 - added meta data contained in a yaml file
 - added style objects located in the style js file to style the output

status: in progress  

tested:
 - headings
 - lists (ordered, unordered, nested unordered)
 - code blocks

todo implement and test: 
 - images
 - thematic breaks
 - fenced code blocks
 - extensions:
   - tables
   - footnotes

## md2jsV4: Performance enhancement

replaced rendering textblocks and paragraphs that have multiple inline 
renderering functions with a single renderTextChildren function

status: to come  

## AstDump

dumps the ast tree of a document to a text file.  

status: working

## testYamlSum

Test program that splits an input file into a Meta (yaml) section, a summary section and a main section.  
This features allows adding meta data to the core markdown file, and a summary paragraph after a 'Summary' heading.  
