FILE_JURY = Main.java
FILE_ANSWER = InventoryManager.java
TARGET_FILENAME = Main

# STOP EDITTING HERE
ZIP_NAME = tc.zip
ZIP_COMMAND = zip
UNZIP_COMMAND = unzip
JFLAGS =
JC = javac
JVM = java
TC_FOLDER = tc
EXT_IN = in
EXT_OUT = out

clean:
	rm -rf **/*.class
	rm -rf **/*.out
	rm -rf **/*.jar
	rm -rf *.class
	rm -rf *.out
	rm -rf *.jar

all: java tc zip

$(TARGET_FILENAME).class: $(FILE_ANSWER) $(FILE_JURY)
	$(JC) $(JFLAGS) $(FILE_JURY) $(FILE_ANSWER)
	jar cfe $(TARGET_FILENAME).jar $(TARGET_FILENAME) *.class

java: $(TARGET_FILENAME).class

tc: $(TC_FOLDER)/*.$(EXT_IN) java
	for filename in $(TC_FOLDER)/*.$(EXT_IN); do \
		$(JVM) $(TARGET_FILENAME) < $$filename > $${filename%.$(EXT_IN)}.$(EXT_OUT); \
	done;

tc.zip: tc Makefile
	-rm tc.zip; \
	$(ZIP_COMMAND) -j $(ZIP_NAME) \
		$(TC_FOLDER)/*.$(EXT_IN) $(TC_FOLDER)/*.$(EXT_OUT) \
		$(FILE_JURY) \
		Makefile;

zip: tc.zip

attachment.zip:
	-rm attachments.zip; \
	$(ZIP_COMMAND) -j attachments.zip \
		attachment/*
