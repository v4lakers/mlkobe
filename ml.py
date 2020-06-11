# Import necessary Libraries
import statistics
import sys
import pandas as pd
from sklearn import preprocessing
from sklearn.linear_model import LogisticRegression
from sklearn.model_selection import cross_val_score
from sklearn.neighbors import KNeighborsClassifier
from sklearn.neural_network import MLPClassifier
from sklearn.tree import DecisionTreeClassifier
from sklearn.svm import SVC

# Read the Data
data = pd.read_csv("data/data.csv")

# Delete Meaningless columns
del data['game_id']
del data['shot_id']
del data['team_id']
del data['game_date']
del data['team_name']

# Categorical Variabels
categorical = ["action_type", "combined_shot_type", "period", "playoffs"
    , "season", "shot_type", "shot_zone_area", "shot_zone_basic", "shot_zone_range", "location", "opponent"]

# Loop through the categorical variables and turn it into dummy data
for column in categorical:
    data = pd.concat([data, pd.get_dummies(data[column], prefix=column)], axis=1)
    del data[column]

# Create Features
features = data.copy()
del features['shot_made_flag']

# Scale the Features
scaler = preprocessing.StandardScaler()
scaler.fit(features)
features = scaler.transform(features)

# Create the Label
label = data['shot_made_flag']


# Get the Parameters from the raw arguments
params = {}
param_name = sys.argv[2]
param_name = param_name.strip("[")
param_name  = param_name.strip("]")
param_name  = param_name.split(" ")
param_vals = sys.argv[3].split("_")

# Save the Parameters to a dictionary
for i in range(len(param_vals)):
    if param_vals[i].isdigit():
        params[param_name[i]] = int(param_vals[i])
    else:
        params[param_name[i]] = param_vals[i]


# Get the model name based on argument
name = sys.argv[1]
if name == "logistic regression":
    model = LogisticRegression(**params)
elif name == "decision tree":
    model = DecisionTreeClassifier(**params)
elif name == "ann":
    model = MLPClassifier(**params)
elif name == "svm":
    model = SVC(**params)
else:
    model = KNeighborsClassifier(**params)

# Run 3-Fold Cross Validation and return results
scores = cross_val_score(model, features, label, cv=3)
print(statistics.mean(scores))

