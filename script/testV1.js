let mdDivObj = {
	typ:'div',
	id: 'mdDiv',
	style: {
		margin: '10px',
		border: '1px dashed blue',
		position: 'relative',
		minHeight: '200px',
	},
};
let mdDiv = azul.addElement(mdDivObj);
let el2= document.createElement("h1");
let el3=document.createTextNode("markdown test file");
// dbg -- el: el3 parent:el2 kind:Heading
el2.appendChild(el3);
// dbg -- el: el2 parent:mdDiv kind:Document
mdDiv.appendChild(el2);
let el4=document.createElement("p");
let el5=document.createTextNode("This is a test file to check the markdown conversion process.");
// dbg -- el: el5 parent:el4 kind:Paragraph
el4.appendChild(el5);
// dbg -- el: el4 parent:mdDiv kind:Document
mdDiv.appendChild(el4);
