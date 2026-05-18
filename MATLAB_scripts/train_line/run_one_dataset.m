% environment setup
model_name = 'model';
load_system(model_name);       
total = 200;

% check that data folders exist and create them if needed
if ~exist('./data/rms', 'dir'), mkdir('./data/rms'); end
if ~exist('./data/sv_and_trip', 'dir'), mkdir('./data/sv_and_trip'); end


% fault generation (every kilometer)
% iterate from 1 km to 199 km to avoid division by zero in line blocks
for fault_dist = 1:1:199
    
    % calculate line variables
    part1 = fault_dist;
    part2 = total - fault_dist;
    
    % print progress to the console
    fprintf('\nFault simulation %d km...', fault_dist);
    
    % run simulation
    % 'SrcWorkspace','current' lets sim see the part1 and part2 variables
    simOut = sim(model_name, 'SrcWorkspace', 'current');
    
    
    % build unique file names
    t_stamp = string(datetime('now'), 'yyyyMMdd_HHmmss');

    % build paths
    filepath_rms = sprintf('./data/rms/%s_%03d_km.csv', t_stamp, fault_dist);
    filepath_sv_trip = sprintf('./data/sv_and_trip/%s_%03d_km.csv', t_stamp, fault_dist);
    
    % save data
    writematrix(simOut.I_U_ABCN_RMS, filepath_rms, 'Delimiter', ';');
    writematrix(simOut.SV_and_Trip, filepath_sv_trip, 'Delimiter', ';');
    
end

fprintf('Dataset generation completed!\n');
