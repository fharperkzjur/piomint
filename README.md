# ai-lab-backend

Restful service for AI lab runs management  

   AI Lab manages runs in two hierarchy , the basic root is called lab which user can construct template data 
according to your needs. Lab is something like a run groups , configure its tags and config data , then run as
many times as necessary , user will see run list behind labs. Besides common run tempate data , labs also 
record user information like user name ,user id , group name ,group id, orgnization name, orgniaztion id etc .
These data apply to all runs which started below this lab .By the way , each run's output directory will also 
be created as needed just under its lab storage directory .

   When lab is created, you can start many kinds of runs belong  to this lab . Different type of runs may 
have different lifecyle and features, but their data structure remains the same . AI Lab implements these
features by `flags` when started , that is to say , all type of runs are no difference with AI Lab core ,but 
their start configuration is different . Different type of runs use different starting code to generate run 
record, thus API is different ,but core controller remains the same . So add a new job type is simple ,only 
need to change start code ,combines some core feature flags . 
  
   AI Lab implement following core features for each run:
   1. 
  
